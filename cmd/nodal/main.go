package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime"
	"syscall"
	"unsafe"

	"nodal/src/autostart"
	"nodal/src/config"
	"nodal/src/dns"
	"nodal/src/elevation"
	"nodal/src/floater"
	"nodal/src/tray"
	"nodal/src/winapi"

	"golang.org/x/sys/windows"
)

func init() {
	// Lock main OS thread for Win32 message loop and GUI calls
	runtime.LockOSThread()
}

var (
	msgHwnd             windows.HWND
	trayMgr             *tray.TrayManager
	floaterWnd          *floater.Floater
	wmTaskbarCreatedMsg uint32
)

const (
	msgClassName = "NodalMsgHostClass"
)

func main() {
	// Parse CLI flags
	isWorker := flag.Bool("worker", false, "Internal elevated worker to apply DNS")
	isRegisterTask := flag.Bool("register-task", false, "Internal elevated registration of scheduled task")
	flag.Parse()

	// 1. Elevated worker execution
	if *isWorker {
		os.Exit(elevation.WorkerMain())
		return
	}

	// 2. Elevated scheduled task registration
	if *isRegisterTask {
		if err := elevation.RegisterTaskAsAdmin(); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
		return
	}

	// 3. Single Instance Guard (Local mutex)
	mutexName, _ := windows.UTF16PtrFromString("Local\\Nodal_SingleInstance_Mutex")
	hMutex, err := windows.CreateMutex(nil, false, mutexName)
	if errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		// An instance is already running in tray
		os.Exit(0)
		return
	}
	if hMutex == 0 {
		os.Exit(1)
		return
	}
	defer windows.CloseHandle(hMutex)

	// 4. Load configuration & sync autostart
	cfg, _ := config.LoadConfig()
	if cfg != nil {
		_ = autostart.Sync(cfg.Autostart)
	}

	// Opt the process into Per-Monitor DPI Awareness v2 so all windows
	// render 1:1 crisp and scaled correctly on high-DPI displays.
	winapi.EnableDpiAwareness()

	// Opt the process into dark-mode awareness so native controls (menus,
	// scrollbars, etc.) follow the system theme dynamically.
	// AllowDark = 1; must be called before any HWND is created.
	winapi.SetPreferredAppMode(1)

	// 5. Register TaskbarCreated message to survive Explorer restarts
	wmTaskbarCreatedMsg = winapi.RegisterWindowMessage("TaskbarCreated")

	// 6. Create hidden message host window
	if err := createMessageHostWindow(); err != nil {
		fmt.Fprintf(os.Stderr, "createMessageHostWindow err: %v\n", err)
		os.Exit(1)
		return
	}

	// 7. Initialise Floater flyout panel
	var errFloater error
	floaterWnd, errFloater = floater.NewFloater(
		func(activeProfile string, uacAlert bool) {
			if trayMgr != nil {
				trayMgr.UpdateState(activeProfile, uacAlert)
			}
		},
		func() {
			_ = config.OpenInEditor()
		},
	)
	if errFloater != nil {
		fmt.Fprintf(os.Stderr, "NewFloater err: %v\n", errFloater)
		os.Exit(1)
		return
	}

	// 8. Initialise Tray Icon
	trayMgr = tray.NewTrayManager(
		msgHwnd,
		func() {
			// Left click: hide context menu, then toggle floater
			if trayMgr != nil {
				trayMgr.HideContextMenu()
			}
			floaterWnd.Toggle()
		},
		func() {
			// Before showing context menu: dismiss floater
			if floaterWnd != nil {
				floaterWnd.Hide()
			}
		},
		func() {
			// Right-click context menu "Open config"
			_ = config.OpenInEditor()
		},
		func() {
			// Right-click context menu "Exit"
			if floaterWnd != nil {
				floaterWnd.Hide()
			}
			if trayMgr != nil {
				trayMgr.Remove()
			}
			os.Exit(0)
		},
	)

	// Detect initial active DNS provider
	initialProfile := "DHCP"
	if cfg != nil {
		initialProfile = dns.DetectCurrentProfile(cfg.Profiles)
	}
	trayMgr.UpdateState(initialProfile, false)

	// Install tray icon
	if err := trayMgr.Add(); err != nil {
		fmt.Fprintf(os.Stderr, "trayMgr.Add err: %v\n", err)
		os.Exit(1)
		return
	}
	defer trayMgr.Remove()

	// 9. Win32 Message Loop
	var msg winapi.MSG
	for {
		r := winapi.GetMessage(&msg, 0, 0, 0)
		if r == 0 || r == -1 {
			break
		}
		winapi.TranslateMessage(&msg)
		winapi.DispatchMessage(&msg)
	}

	// Clean shutdown
	floaterWnd.Hide()
	trayMgr.Remove()
}

func createMessageHostWindow() error {
	pClassName, _ := windows.UTF16PtrFromString(msgClassName)

	var wc winapi.WNDCLASSEXW
	wc.CbSize = uint32(unsafe.Sizeof(wc))
	wc.LpfnWndProc = syscall.NewCallback(msgWndProc)
	wc.LpszClassName = pClassName

	_, err := winapi.RegisterClassEx(&wc)
	if err != nil && !errors.Is(err, windows.ERROR_CLASS_ALREADY_EXISTS) {
		return err
	}

	hwnd, err := winapi.CreateWindowEx(
		0,
		pClassName,
		pClassName,
		winapi.WS_POPUP, // standard top-level hidden window required by Shell_NotifyIcon
		0, 0, 0, 0,
		0,
		0, 0, 0,
	)
	if err != nil {
		return err
	}

	msgHwnd = hwnd
	return nil
}

func msgWndProc(hwnd windows.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case winapi.WM_USER + 1:
		// Tray icon mouse events
		if trayMgr != nil {
			trayMgr.HandleCallback(lParam)
		}
		return 0

	case winapi.WM_DESTROY:
		winapi.PostQuitMessage(0)
		return 0

	default:
		// Taskbar recreated (explorer.exe crash or restart)
		if wmTaskbarCreatedMsg != 0 && msg == wmTaskbarCreatedMsg {
			if trayMgr != nil {
				_ = trayMgr.Add()
			}
			return 0
		}
	}
	return winapi.DefWindowProc(hwnd, msg, wParam, lParam)
}
