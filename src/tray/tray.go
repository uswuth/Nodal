package tray

import (
	"fmt"
	"sync"
	"time"
	"unsafe"

	"nodal/src/config"
	"nodal/src/menu"
	"nodal/src/winapi"

	"golang.org/x/sys/windows"
)

type TrayManager struct {
	mu                  sync.Mutex
	hwnd                windows.HWND
	hIcon               windows.Handle
	currentDNS          string
	uacAlert            bool
	lastClickTime       time.Time
	contextMenu         *menu.MenuFlyout
	onLeftClick         func()
	onBeforeContextMenu func()
	onOpenConfig        func()
	onCleanUninstall    func()
	onExit              func()
}

func NewTrayManager(hwnd windows.HWND, onLeftClick, onBeforeContextMenu, onOpenConfig, onCleanUninstall, onExit func()) *TrayManager {
	tm := &TrayManager{
		hwnd:                hwnd,
		currentDNS:          "DHCP",
		onLeftClick:         onLeftClick,
		onBeforeContextMenu: onBeforeContextMenu,
		onOpenConfig:        onOpenConfig,
		onCleanUninstall:    onCleanUninstall,
		onExit:              onExit,
	}

	cm, err := menu.NewMenuFlyout(
		func() {
			winapi.DnsFlushResolverCache()
		},
		func() {
			if tm.onOpenConfig != nil {
				tm.onOpenConfig()
			} else {
				_ = config.OpenInEditor()
			}
		},
		func() {
			if tm.onCleanUninstall != nil {
				tm.onCleanUninstall()
			}
		},
		func() {
			if tm.onExit != nil {
				tm.onExit()
			}
		},
	)
	if err == nil {
		tm.contextMenu = cm
	}

	return tm
}

// Add installs the icon into the Windows 11 notification area with GUID and tooltip
func (t *TrayManager) Add() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	icon, err := CreateNodalIcon(0, t.uacAlert)
	if err != nil {
		return err
	}
	t.hIcon = icon

	// Clean any previous stale icon instance for this HWND/UID or GUID
	var nidDel winapi.NOTIFYICONDATAW
	nidDel.CbSize = uint32(unsafe.Sizeof(nidDel))
	nidDel.HWnd = t.hwnd
	nidDel.UID = 1
	winapi.Shell_NotifyIcon(winapi.NIM_DELETE, &nidDel)

	var nidDelGuid winapi.NOTIFYICONDATAW
	nidDelGuid.CbSize = uint32(unsafe.Sizeof(nidDelGuid))
	nidDelGuid.UFlags = winapi.NIF_GUID
	nidDelGuid.GuidItem = winapi.NodalTrayGUID
	winapi.Shell_NotifyIcon(winapi.NIM_DELETE, &nidDelGuid)

	var nid winapi.NOTIFYICONDATAW
	nid.CbSize = uint32(unsafe.Sizeof(nid))
	nid.HWnd = t.hwnd
	nid.UID = 1
	nid.UFlags = winapi.NIF_MESSAGE | winapi.NIF_ICON | winapi.NIF_TIP | winapi.NIF_SHOWTIP | winapi.NIF_GUID
	nid.UCallbackMessage = winapi.WM_USER + 1
	nid.HIcon = t.hIcon
	nid.GuidItem = winapi.NodalTrayGUID

	tipText := t.formatTooltip()
	tipW, _ := windows.UTF16FromString(tipText)
	copy(nid.SzTip[:], tipW)

	usedGuid := true
	if !winapi.Shell_NotifyIcon(winapi.NIM_ADD, &nid) {
		// Fallback without NIF_GUID if GUID was previously cached to a different path
		usedGuid = false
		nid.UFlags = winapi.NIF_MESSAGE | winapi.NIF_ICON | winapi.NIF_TIP | winapi.NIF_SHOWTIP
		nid.GuidItem = windows.GUID{}
		if !winapi.Shell_NotifyIcon(winapi.NIM_ADD, &nid) {
			winapi.Shell_NotifyIcon(winapi.NIM_DELETE, &nid)
			if !winapi.Shell_NotifyIcon(winapi.NIM_ADD, &nid) {
				return fmt.Errorf("Shell_NotifyIcon NIM_ADD failed")
			}
		}
	}

	// Set NOTIFYICON_VERSION_4 behavior
	var nidVer winapi.NOTIFYICONDATAW
	nidVer.CbSize = uint32(unsafe.Sizeof(nidVer))
	nidVer.HWnd = t.hwnd
	nidVer.UID = 1
	if usedGuid {
		nidVer.UFlags = winapi.NIF_GUID
		nidVer.GuidItem = winapi.NodalTrayGUID
	}
	nidVer.TimeoutOrVersion = winapi.NOTIFYICON_VERSION_4
	winapi.Shell_NotifyIcon(winapi.NIM_SETVERSION, &nidVer)

	return nil
}

// UpdateState updates the current DNS provider name and UAC alert status
func (t *TrayManager) UpdateState(provider string, uacAlert bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	iconChanged := t.uacAlert != uacAlert
	t.currentDNS = provider
	t.uacAlert = uacAlert

	var nid winapi.NOTIFYICONDATAW
	nid.CbSize = uint32(unsafe.Sizeof(nid))
	nid.HWnd = t.hwnd
	nid.UID = 1
	nid.UFlags = winapi.NIF_TIP | winapi.NIF_SHOWTIP | winapi.NIF_GUID
	nid.GuidItem = winapi.NodalTrayGUID

	tipText := t.formatTooltip()
	tipW, _ := windows.UTF16FromString(tipText)
	copy(nid.SzTip[:], tipW)

	if iconChanged {
		if t.hIcon != 0 {
			winapi.DestroyIcon(t.hIcon)
		}
		newIcon, err := CreateNodalIcon(0, t.uacAlert)
		if err == nil {
			t.hIcon = newIcon
			nid.HIcon = newIcon
			nid.UFlags |= winapi.NIF_ICON
		}
	}

	if !winapi.Shell_NotifyIcon(winapi.NIM_MODIFY, &nid) {
		nid.UFlags &^= winapi.NIF_GUID
		winapi.Shell_NotifyIcon(winapi.NIM_MODIFY, &nid)
	}
}

func (t *TrayManager) formatTooltip() string {
	base := fmt.Sprintf("Nodal — %s", t.currentDNS)
	if t.uacAlert {
		base += " (Admin required)"
	}
	return base
}

// Remove removes the icon from the notification area
func (t *TrayManager) Remove() {
	t.mu.Lock()
	defer t.mu.Unlock()

	var nid winapi.NOTIFYICONDATAW
	nid.CbSize = uint32(unsafe.Sizeof(nid))
	nid.HWnd = t.hwnd
	nid.UID = 1
	nid.UFlags = winapi.NIF_GUID
	nid.GuidItem = winapi.NodalTrayGUID

	if !winapi.Shell_NotifyIcon(winapi.NIM_DELETE, &nid) {
		nid.UFlags = 0
		winapi.Shell_NotifyIcon(winapi.NIM_DELETE, &nid)
	}

	if t.hIcon != 0 {
		winapi.DestroyIcon(t.hIcon)
		t.hIcon = 0
	}
}

// HandleCallback processes mouse messages sent from the tray icon
func (t *TrayManager) HandleCallback(lParam uintptr) {
	msg := uint32(lParam & 0xFFFF)
	switch msg {
	case winapi.WM_LBUTTONUP, winapi.NIN_SELECT:
		t.mu.Lock()
		now := time.Now()
		if now.Sub(t.lastClickTime) < 400*time.Millisecond {
			t.mu.Unlock()
			return
		}
		t.lastClickTime = now
		t.mu.Unlock()

		if t.onLeftClick != nil {
			t.onLeftClick()
		}

	case winapi.WM_RBUTTONUP, winapi.WM_CONTEXTMENU:
		t.mu.Lock()
		now := time.Now()
		if now.Sub(t.lastClickTime) < 400*time.Millisecond {
			t.mu.Unlock()
			return
		}
		t.lastClickTime = now
		t.mu.Unlock()

		t.showContextMenu()
	}

}

func (t *TrayManager) HideContextMenu() {
	if t.contextMenu != nil {
		t.contextMenu.Hide()
	}
}

func (t *TrayManager) showContextMenu() {
	if t.onBeforeContextMenu != nil {
		t.onBeforeContextMenu()
	}

	var pt winapi.POINT
	winapi.GetCursorPos(&pt)

	if t.contextMenu != nil {
		t.contextMenu.Show(pt)
		return
	}

	t.fallbackNativeMenu(pt)
}

func (t *TrayManager) fallbackNativeMenu(pt winapi.POINT) {
	hMenu := winapi.CreatePopupMenu()
	if hMenu == 0 {
		return
	}
	defer winapi.DestroyMenu(hMenu)

	const (
		idFlushDNS       = 1001
		idOpenConfig     = 1002
		idExit           = 1003
		idCleanUninstall = 1004
	)

	winapi.AppendMenu(hMenu, winapi.MF_STRING, idFlushDNS, "Flush DNS cache")
	winapi.AppendMenu(hMenu, winapi.MF_STRING, idOpenConfig, "Open configuration")
	winapi.AppendMenu(hMenu, winapi.MF_SEPARATOR, 0, "")
	winapi.AppendMenu(hMenu, winapi.MF_STRING, idCleanUninstall, "Clean uninstall")
	winapi.AppendMenu(hMenu, winapi.MF_STRING, idExit, "Exit Nodal")

	// Apply dark/light theme to owner window so Windows renders native Win11 popup menu
	isDark := !winapi.GetAppsUseLightTheme()
	if isDark {
		themeStr, _ := windows.UTF16PtrFromString("DarkMode_Explorer")
		winapi.SetWindowTheme(t.hwnd, themeStr, nil)
		winapi.AllowDarkModeForWindow(t.hwnd, true)
	} else {
		themeStr, _ := windows.UTF16PtrFromString("Explorer")
		winapi.SetWindowTheme(t.hwnd, themeStr, nil)
		winapi.AllowDarkModeForWindow(t.hwnd, false)
	}

	winapi.SetForegroundWindow(t.hwnd)

	cmd := winapi.TrackPopupMenu(
		hMenu,
		winapi.TPM_RIGHTBUTTON|winapi.TPM_RETURNCMD,
		pt.X, pt.Y,
		0,
		t.hwnd,
		nil,
	)
	winapi.PostMessage(t.hwnd, winapi.WM_NULL, 0, 0)

	switch cmd {
	case idFlushDNS:
		winapi.DnsFlushResolverCache()
	case idOpenConfig:
		if t.onOpenConfig != nil {
			t.onOpenConfig()
		} else {
			_ = config.OpenInEditor()
		}
	case idCleanUninstall:
		// Run off the message thread: the flow waits for UAC and for the elevated instance.
		if handler := t.onCleanUninstall; handler != nil {
			go handler()
		}
	case idExit:
		if t.onExit != nil {
			t.onExit()
		}
	}
}

