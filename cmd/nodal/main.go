//go:generate goversioninfo

package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"nodal/src/autostart"
	"nodal/src/cleanup"
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

	// elevatedWaitTimeout bounds how long an instance waits for its elevated counterpart,
	// so a crashed helper can never block the uninstaller or the tray application forever.
	elevatedWaitTimeout = 60 * time.Second
)

func main() {
	// Parse CLI flags
	isWorker := flag.Bool("worker", false, "Internal elevated worker to apply DNS")
	isRegisterTask := flag.Bool("register-task", false, "Internal elevated registration of scheduled task")
	enableAutostart := flag.Bool("enable-autostart", false, "Enable launch on Windows sign-in, then exit")
	disableAutostart := flag.Bool("disable-autostart", false, "Disable launch on Windows sign-in, then exit")
	resetDNS := flag.Bool("reset-dns", false, "Restore automatic (DHCP) DNS on all adapters, then exit")
	cleanUninstall := flag.Bool("clean-uninstall", false, "Remove Nodal, its settings and its DNS changes, then exit")
	assumeYes := flag.Bool("yes", false, "Skip the clean uninstall confirmation prompt")
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

	// 3. Installer-driven autostart preference (persisted so it survives the first launch)
	if *enableAutostart || *disableAutostart {
		os.Exit(setAutostartPreference(*enableAutostart))
		return
	}

	// 4. Restore automatic (DHCP-provided) DNS on every adapter
	if *resetDNS {
		os.Exit(runResetDNS())
		return
	}

	// 5. Clean uninstall: reverse every Nodal change, then remove the application
	if *cleanUninstall {
		os.Exit(runCleanUninstall(*assumeYes))
		return
	}

	// 6. Single Instance Guard (Local mutex)
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

	// 7. Load configuration & sync autostart
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

	// 8. Register TaskbarCreated message to survive Explorer restarts
	wmTaskbarCreatedMsg = winapi.RegisterWindowMessage("TaskbarCreated")

	// 9. Create hidden message host window
	if err := createMessageHostWindow(); err != nil {
		fmt.Fprintf(os.Stderr, "createMessageHostWindow err: %v\n", err)
		os.Exit(1)
		return
	}

	// 10. Initialise Floater flyout panel
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

	// 11. Initialise Tray Icon
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
			// Right-click context menu "Clean uninstall"
			runCleanUninstall(false)
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

	// 12. Win32 Message Loop
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

// setAutostartPreference persists the autostart flag to config.toml and mirrors it
// into the registry Run key, so an installer-selected choice survives the first launch.
func setAutostartPreference(enabled bool) int {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config err: %v\n", err)
	}
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	cfg.Autostart = enabled
	if err := config.SaveConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "save config err: %v\n", err)
		return 1
	}
	if err := autostart.Sync(enabled); err != nil {
		fmt.Fprintf(os.Stderr, "sync autostart err: %v\n", err)
		return 1
	}
	return 0
}

// runResetDNS restores automatic (DHCP-provided) DNS on every adapter, elevating when the caller
// is not already running with administrative rights. The uninstaller invokes it before deleting
// any files, so the elevated work is always finished when this process exits.
func runResetDNS() int {
	if !elevation.IsElevated() {
		if err := elevation.RunSelfElevatedAndWait("--reset-dns", elevatedWaitTimeout); err != nil {
			fmt.Fprintf(os.Stderr, "elevation err: %v\n", err)
			return 2
		}
		return 0
	}

	err := dns.ResetAllToDHCP()
	elevation.SignalElevatedDone()

	if err != nil {
		fmt.Fprintf(os.Stderr, "reset DNS err: %v\n", err)
		return 1
	}
	return 0
}

// runCleanUninstall reverses every change Nodal made to the machine and then removes the
// application itself. Work that needs administrative rights is handed to an elevated instance.
func runCleanUninstall(assumeYes bool) int {
	if !assumeYes && !confirmCleanUninstall() {
		return 0
	}

	if !elevation.IsElevated() {
		err := elevation.RunSelfElevatedAndWait("--clean-uninstall --yes", elevatedWaitTimeout)
		if err != nil && !errors.Is(err, elevation.ErrElevationTimeout) {
			// The consent prompt was declined or elevation failed, so nothing was changed.
			if errors.Is(err, elevation.ErrUACCancelled) {
				showCleanUninstallMessage("Clean uninstall cancelled.\r\n\r\nNothing was changed.")
			} else {
				showCleanUninstallMessage("Clean uninstall could not start.\r\n\r\n" + err.Error())
			}
			return 2
		}

		// The elevated instance either finished or runs under another administrator account, whose
		// temporary folder this process cannot observe. Step aside either way so the uninstaller
		// can remove the files.
		if trayMgr != nil {
			trayMgr.Remove()
		}
		return 0
	}

	report := cleanup.Run(true)
	if trayMgr != nil {
		trayMgr.Remove()
	}

	// Release the waiting instance before showing anything, so it can exit promptly.
	elevation.SignalElevatedDone()

	if message := report.UserMessage(); message != "" {
		showCleanUninstallMessage(message)
	}

	if report.UninstallerPath != "" {
		if err := cleanup.LaunchUninstaller(report.UninstallerPath); err != nil {
			showCleanUninstallMessage("The bundled uninstaller could not be started.\r\n\r\n" + err.Error())
			return 1
		}
	}
	return report.ExitCode()
}

// confirmCleanUninstall asks the user to approve the destructive clean uninstall.
func confirmCleanUninstall() bool {
	message := "Clean uninstall will:\r\n\r\n" +
		"  -  restore the automatic (DHCP) DNS resolvers on all network adapters\r\n" +
		"  -  remove the privileged worker scheduled task\r\n" +
		"  -  remove the Windows startup entry\r\n" +
		"  -  delete the Nodal settings, saved DNS presets and temporary files\r\n" +
		"  -  uninstall the application itself\r\n\r\n" +
		"Nothing else on this PC is modified."

	if !elevation.IsElevated() {
		message += "\r\n\r\nWindows will next ask for administrator approval."
	}
	message += "\r\n\r\nContinue?"

	flags := uint32(winapi.MB_YESNO | winapi.MB_ICONWARNING | winapi.MB_SETFOREGROUND | winapi.MB_TOPMOST)
	return winapi.MessageBox("Nodal - Clean uninstall", message, flags) == winapi.IDYES
}

func showCleanUninstallMessage(message string) {
	flags := uint32(winapi.MB_OK | winapi.MB_ICONINFORMATION | winapi.MB_SETFOREGROUND | winapi.MB_TOPMOST)
	winapi.MessageBox("Nodal - Clean uninstall", message, flags)
}
