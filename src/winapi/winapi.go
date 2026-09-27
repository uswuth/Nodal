package winapi

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var (
	// DLLs
	user32   = windows.NewLazyDLL("user32.dll")
	shell32  = windows.NewLazyDLL("shell32.dll")
	dwmapi   = windows.NewLazyDLL("dwmapi.dll")
	gdi32    = windows.NewLazyDLL("gdi32.dll")
	iphlpapi = windows.NewLazyDLL("iphlpapi.dll")
	dnsapi   = windows.NewLazyDLL("dnsapi.dll")
	kernel32 = windows.NewLazyDLL("kernel32.dll")
	ntdll    = windows.NewLazyDLL("ntdll.dll")
	uxtheme  = windows.NewLazyDLL("uxtheme.dll")

	// user32 procs
	procRegisterClassExW      = user32.NewProc("RegisterClassExW")
	procCreateWindowExW        = user32.NewProc("CreateWindowExW")
	procDefWindowProcW         = user32.NewProc("DefWindowProcW")
	procDestroyWindow          = user32.NewProc("DestroyWindow")
	procPostQuitMessage        = user32.NewProc("PostQuitMessage")
	procPostMessageW           = user32.NewProc("PostMessageW")
	procShowWindow             = user32.NewProc("ShowWindow")
	procSetWindowPos           = user32.NewProc("SetWindowPos")
	procGetWindowRect          = user32.NewProc("GetWindowRect")
	procGetClientRect          = user32.NewProc("GetClientRect")
	procGetCursorPos           = user32.NewProc("GetCursorPos")
	procSetForegroundWindow    = user32.NewProc("SetForegroundWindow")
	procGetForegroundWindow    = user32.NewProc("GetForegroundWindow")
	procTrackMouseEvent        = user32.NewProc("TrackMouseEvent")
	procBeginPaint             = user32.NewProc("BeginPaint")
	procEndPaint               = user32.NewProc("EndPaint")
	procInvalidateRect         = user32.NewProc("InvalidateRect")
	procFillRect               = user32.NewProc("FillRect")
	procDrawTextW              = user32.NewProc("DrawTextW")
	procCreatePopupMenu        = user32.NewProc("CreatePopupMenu")
	procAppendMenuW            = user32.NewProc("AppendMenuW")
	procTrackPopupMenu         = user32.NewProc("TrackPopupMenu")
	procDestroyMenu            = user32.NewProc("DestroyMenu")
	procGetSystemMetrics       = user32.NewProc("GetSystemMetrics")
	procGetDpiForWindow        = user32.NewProc("GetDpiForWindow")
	procMonitorFromPoint       = user32.NewProc("MonitorFromPoint")
	procGetMonitorInfoW        = user32.NewProc("GetMonitorInfoW")
	procFindWindowW            = user32.NewProc("FindWindowW")
	procFindWindowExW          = user32.NewProc("FindWindowExW")
	procCreateIconIndirect     = user32.NewProc("CreateIconIndirect")
	procDestroyIcon            = user32.NewProc("DestroyIcon")
	procRegisterWindowMessageW = user32.NewProc("RegisterWindowMessageW")
	procGetMessageW            = user32.NewProc("GetMessageW")
	procTranslateMessage       = user32.NewProc("TranslateMessage")
	procDispatchMessageW       = user32.NewProc("DispatchMessageW")
	procLoadCursorW            = user32.NewProc("LoadCursorW")
	procSetTimer               = user32.NewProc("SetTimer")
	procKillTimer              = user32.NewProc("KillTimer")

	// shell32 procs
	procShell_NotifyIconW       = shell32.NewProc("Shell_NotifyIconW")
	procShell_NotifyIconGetRect = shell32.NewProc("Shell_NotifyIconGetRect")
	procSHAppBarMessage         = shell32.NewProc("SHAppBarMessage")

	// dwmapi procs
	procDwmSetWindowAttribute = dwmapi.NewProc("DwmSetWindowAttribute")
	procDwmGetColorizationColor = dwmapi.NewProc("DwmGetColorizationColor")

	// gdi32 procs
	procCreateBitmap           = gdi32.NewProc("CreateBitmap")
	procCreateCompatibleDC     = gdi32.NewProc("CreateCompatibleDC")
	procCreateCompatibleBitmap = gdi32.NewProc("CreateCompatibleBitmap")
	procCreateDIBSection       = gdi32.NewProc("CreateDIBSection")
	procSelectObject           = gdi32.NewProc("SelectObject")
	procDeleteObject           = gdi32.NewProc("DeleteObject")
	procDeleteDC               = gdi32.NewProc("DeleteDC")
	procBitBlt                 = gdi32.NewProc("BitBlt")
	procSetBkMode              = gdi32.NewProc("SetBkMode")
	procSetTextColor           = gdi32.NewProc("SetTextColor")
	procCreateSolidBrush       = gdi32.NewProc("CreateSolidBrush")
	procCreateFontW            = gdi32.NewProc("CreateFontW")
	procCreateRoundRectRgn     = gdi32.NewProc("CreateRoundRectRgn")
	procCreateRectRgn          = gdi32.NewProc("CreateRectRgn")
	procSelectClipRgn          = gdi32.NewProc("SelectClipRgn")
	procFillRgn                = gdi32.NewProc("FillRgn")
	procFrameRgn               = gdi32.NewProc("FrameRgn")


	// iphlpapi & dnsapi procs
	procSetInterfaceDnsSettings = iphlpapi.NewProc("SetInterfaceDnsSettings")
	procGetInterfaceDnsSettings = iphlpapi.NewProc("GetInterfaceDnsSettings")
	procDnsFlushResolverCache   = dnsapi.NewProc("DnsFlushResolverCache")

	// ntdll procs
	procRtlGetVersion = ntdll.NewProc("RtlGetVersion")

	// uxtheme procs
	procSetWindowTheme = uxtheme.NewProc("SetWindowTheme")
)

// Win32 Wrappers

func RegisterClassEx(wc *WNDCLASSEXW) (uint16, error) {
	r1, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(wc)))
	if r1 == 0 {
		return 0, err
	}
	return uint16(r1), nil
}

func CreateWindowEx(exStyle uint32, className, windowName *uint16, style uint32, x, y, width, height int32, parent windows.HWND, menu windows.Handle, instance windows.Handle, param uintptr) (windows.HWND, error) {
	r1, _, err := procCreateWindowExW.Call(
		uintptr(exStyle),
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(windowName)),
		uintptr(style),
		uintptr(x),
		uintptr(y),
		uintptr(width),
		uintptr(height),
		uintptr(parent),
		uintptr(menu),
		uintptr(instance),
		param,
	)
	if r1 == 0 {
		return 0, err
	}
	return windows.HWND(r1), nil
}

func DefWindowProc(hwnd windows.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	r1, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return r1
}

func DestroyWindow(hwnd windows.HWND) error {
	r1, _, err := procDestroyWindow.Call(uintptr(hwnd))
	if r1 == 0 {
		return err
	}
	return nil
}

func PostQuitMessage(exitCode int32) {
	procPostQuitMessage.Call(uintptr(exitCode))
}

func PostMessage(hwnd windows.HWND, msg uint32, wParam, lParam uintptr) bool {
	r1, _, _ := procPostMessageW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return r1 != 0
}

func ShowWindow(hwnd windows.HWND, cmdShow int32) bool {
	r1, _, _ := procShowWindow.Call(uintptr(hwnd), uintptr(cmdShow))
	return r1 != 0
}

func SetWindowPos(hwnd windows.HWND, insertAfter uintptr, x, y, cx, cy int32, flags uint32) bool {
	r1, _, _ := procSetWindowPos.Call(
		uintptr(hwnd),
		insertAfter,
		uintptr(x),
		uintptr(y),
		uintptr(cx),
		uintptr(cy),
		uintptr(flags),
	)
	return r1 != 0
}

func GetWindowRect(hwnd windows.HWND, rect *RECT) bool {
	r1, _, _ := procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(rect)))
	return r1 != 0
}

func GetClientRect(hwnd windows.HWND, rect *RECT) bool {
	r1, _, _ := procGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(rect)))
	return r1 != 0
}

func GetCursorPos(pt *POINT) bool {
	r1, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(pt)))
	return r1 != 0
}

func SetForegroundWindow(hwnd windows.HWND) bool {
	r1, _, _ := procSetForegroundWindow.Call(uintptr(hwnd))
	return r1 != 0
}

func GetForegroundWindow() windows.HWND {
	r1, _, _ := procGetForegroundWindow.Call()
	return windows.HWND(r1)
}

func TrackMouse(tme *TRACKMOUSEEVENT) bool {
	r1, _, _ := procTrackMouseEvent.Call(uintptr(unsafe.Pointer(tme)))
	return r1 != 0
}

func BeginPaint(hwnd windows.HWND, ps *PAINTSTRUCT) windows.Handle {
	r1, _, _ := procBeginPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(ps)))
	return windows.Handle(r1)
}

func EndPaint(hwnd windows.HWND, ps *PAINTSTRUCT) {
	procEndPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(ps)))
}

func InvalidateRect(hwnd windows.HWND, rect *RECT, erase bool) bool {
	var eraseVal uintptr
	if erase {
		eraseVal = 1
	}
	r1, _, _ := procInvalidateRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(rect)), eraseVal)
	return r1 != 0
}

func FillRect(hdc windows.Handle, rc *RECT, hbr windows.Handle) int32 {
	r1, _, _ := procFillRect.Call(uintptr(hdc), uintptr(unsafe.Pointer(rc)), uintptr(hbr))
	return int32(r1)
}

func DrawText(hdc windows.Handle, text string, rc *RECT, format uint32) int32 {
	utf16Text, err := windows.UTF16FromString(text)
	if err != nil {
		return 0
	}
	r1, _, _ := procDrawTextW.Call(
		uintptr(hdc),
		uintptr(unsafe.Pointer(&utf16Text[0])),
		uintptr(len(utf16Text)-1),
		uintptr(unsafe.Pointer(rc)),
		uintptr(format),
	)
	return int32(r1)
}

func RegisterWindowMessage(name string) uint32 {
	pName, _ := windows.UTF16PtrFromString(name)
	r, _, _ := procRegisterWindowMessageW.Call(uintptr(unsafe.Pointer(pName)))
	return uint32(r)
}

func GetMessage(msg *MSG, hwnd windows.HWND, filterMin, filterMax uint32) int32 {
	r, _, _ := procGetMessageW.Call(
		uintptr(unsafe.Pointer(msg)),
		uintptr(hwnd),
		uintptr(filterMin),
		uintptr(filterMax),
	)
	return int32(r)
}

func TranslateMessage(msg *MSG) bool {
	r, _, _ := procTranslateMessage.Call(uintptr(unsafe.Pointer(msg)))
	return r != 0
}

func DispatchMessage(msg *MSG) uintptr {
	r, _, _ := procDispatchMessageW.Call(uintptr(unsafe.Pointer(msg)))
	return r
}

func LoadCursor(cursorID uint32) uintptr {
	r, _, _ := procLoadCursorW.Call(0, uintptr(cursorID))
	return r
}

// Shell & Tray

func Shell_NotifyIcon(message uint32, data *NOTIFYICONDATAW) bool {
	r1, _, _ := procShell_NotifyIconW.Call(uintptr(message), uintptr(unsafe.Pointer(data)))
	return r1 != 0
}

func Shell_NotifyIconGetRect(ident *NOTIFYICONIDENTIFIER, rect *RECT) error {
	r1, _, err := procShell_NotifyIconGetRect.Call(
		uintptr(unsafe.Pointer(ident)),
		uintptr(unsafe.Pointer(rect)),
	)
	if r1 != 0 {
		return fmt.Errorf("Shell_NotifyIconGetRect failed: %v", err)
	}
	return nil
}

// GetTrayIconRect queries the exact screen coordinates of Nodal's tray icon.
func GetTrayIconRect(hwnd windows.HWND) (RECT, bool) {
	var ident NOTIFYICONIDENTIFIER
	ident.CbSize = uint32(unsafe.Sizeof(ident))
	ident.HWnd = hwnd
	ident.GuidItem = NodalTrayGUID
	var rc RECT
	if err := Shell_NotifyIconGetRect(&ident, &rc); err == nil && rc.Width() > 0 && rc.Height() > 0 {
		return rc, true
	}
	// Fallback with UID 1
	ident.GuidItem = windows.GUID{}
	ident.UID = 1
	if err := Shell_NotifyIconGetRect(&ident, &rc); err == nil && rc.Width() > 0 && rc.Height() > 0 {
		return rc, true
	}
	return RECT{}, false
}

func SHAppBarMessage(message uint32, data *APPBARDATA) uintptr {
	r1, _, _ := procSHAppBarMessage.Call(uintptr(message), uintptr(unsafe.Pointer(data)))
	return r1
}

func MonitorFromPoint(pt POINT, flags uint32) windows.Handle {
	// In Win64 ABI, 8-byte struct POINT { LONG x, y; } passed by value is packed into a single 64-bit integer register (RCX)
	packedPt := uintptr(uint32(pt.X)) | (uintptr(uint32(pt.Y)) << 32)
	r1, _, _ := procMonitorFromPoint.Call(packedPt, uintptr(flags))
	return windows.Handle(r1)
}

func GetMonitorInfo(hMonitor windows.Handle, mi *MONITORINFO) bool {
	mi.CbSize = uint32(unsafe.Sizeof(*mi))
	r1, _, _ := procGetMonitorInfoW.Call(uintptr(hMonitor), uintptr(unsafe.Pointer(mi)))
	return r1 != 0
}

func FindWindow(className, windowName string) windows.HWND {
	var pClass, pWindow *uint16
	if className != "" {
		pClass, _ = windows.UTF16PtrFromString(className)
	}
	if windowName != "" {
		pWindow, _ = windows.UTF16PtrFromString(windowName)
	}
	r1, _, _ := procFindWindowW.Call(uintptr(unsafe.Pointer(pClass)), uintptr(unsafe.Pointer(pWindow)))
	return windows.HWND(r1)
}

func FindWindowEx(parent, childAfter windows.HWND, className, windowName string) windows.HWND {
	var pClass, pWindow *uint16
	if className != "" {
		pClass, _ = windows.UTF16PtrFromString(className)
	}
	if windowName != "" {
		pWindow, _ = windows.UTF16PtrFromString(windowName)
	}
	r1, _, _ := procFindWindowExW.Call(
		uintptr(parent),
		uintptr(childAfter),
		uintptr(unsafe.Pointer(pClass)),
		uintptr(unsafe.Pointer(pWindow)),
	)
	return windows.HWND(r1)
}

func CreateIconIndirect(ii *ICONINFO) (windows.Handle, error) {
	r1, _, err := procCreateIconIndirect.Call(uintptr(unsafe.Pointer(ii)))
	if r1 == 0 {
		return 0, err
	}
	return windows.Handle(r1), nil
}

func DestroyIcon(hIcon windows.Handle) bool {
	r1, _, _ := procDestroyIcon.Call(uintptr(hIcon))
	return r1 != 0
}

// GDI Helpers

func CreateBitmap(nWidth, nHeight int32, nPlanes, nBitCount uint32, lpBits unsafe.Pointer) windows.Handle {
	r1, _, _ := procCreateBitmap.Call(
		uintptr(nWidth),
		uintptr(nHeight),
		uintptr(nPlanes),
		uintptr(nBitCount),
		uintptr(lpBits),
	)
	return windows.Handle(r1)
}

func CreateCompatibleDC(hdc windows.Handle) windows.Handle {
	r1, _, _ := procCreateCompatibleDC.Call(uintptr(hdc))
	return windows.Handle(r1)
}

func CreateCompatibleBitmap(hdc windows.Handle, cx, cy int32) windows.Handle {
	r1, _, _ := procCreateCompatibleBitmap.Call(uintptr(hdc), uintptr(cx), uintptr(cy))
	return windows.Handle(r1)
}

func CreateDIBSection(hdc windows.Handle, pbmi *BITMAPINFO, usage uint32, ppvBits **byte, hSection windows.Handle, offset uint32) windows.Handle {
	r1, _, _ := procCreateDIBSection.Call(
		uintptr(hdc),
		uintptr(unsafe.Pointer(pbmi)),
		uintptr(usage),
		uintptr(unsafe.Pointer(ppvBits)),
		uintptr(hSection),
		uintptr(offset),
	)
	return windows.Handle(r1)
}

func SelectObject(hdc windows.Handle, obj windows.Handle) windows.Handle {
	r1, _, _ := procSelectObject.Call(uintptr(hdc), uintptr(obj))
	return windows.Handle(r1)
}

func DeleteObject(obj windows.Handle) bool {
	r1, _, _ := procDeleteObject.Call(uintptr(obj))
	return r1 != 0
}

func DeleteDC(hdc windows.Handle) bool {
	r1, _, _ := procDeleteDC.Call(uintptr(hdc))
	return r1 != 0
}

func BitBlt(hdcDest windows.Handle, nXDest, nYDest, nWidth, nHeight int32, hdcSrc windows.Handle, nXSrc, nYSrc int32, dwRop uint32) bool {
	r1, _, _ := procBitBlt.Call(
		uintptr(hdcDest),
		uintptr(nXDest),
		uintptr(nYDest),
		uintptr(nWidth),
		uintptr(nHeight),
		uintptr(hdcSrc),
		uintptr(nXSrc),
		uintptr(nYSrc),
		uintptr(dwRop),
	)
	return r1 != 0
}

func SetBkMode(hdc windows.Handle, mode int32) int32 {
	r1, _, _ := procSetBkMode.Call(uintptr(hdc), uintptr(mode))
	return int32(r1)
}

func SetTextColor(hdc windows.Handle, color uint32) uint32 {
	r1, _, _ := procSetTextColor.Call(uintptr(hdc), uintptr(color))
	return uint32(r1)
}

func CreateSolidBrush(color uint32) windows.Handle {
	r1, _, _ := procCreateSolidBrush.Call(uintptr(color))
	return windows.Handle(r1)
}

func CreateFont(height, width, escapement, orientation, weight int32, italic, underline, strikeOut uint32, charSet, outPrecision, clipPrecision, quality, pitchAndFamily uint32, faceName string) windows.Handle {
	pFace, _ := windows.UTF16PtrFromString(faceName)
	r1, _, _ := procCreateFontW.Call(
		uintptr(height),
		uintptr(width),
		uintptr(escapement),
		uintptr(orientation),
		uintptr(weight),
		uintptr(italic),
		uintptr(underline),
		uintptr(strikeOut),
		uintptr(charSet),
		uintptr(outPrecision),
		uintptr(clipPrecision),
		uintptr(quality),
		uintptr(pitchAndFamily),
		uintptr(unsafe.Pointer(pFace)),
	)
	return windows.Handle(r1)
}

func CreateRoundRectRgn(x1, y1, x2, y2, w, h int32) windows.Handle {
	r1, _, _ := procCreateRoundRectRgn.Call(
		uintptr(x1), uintptr(y1), uintptr(x2), uintptr(y2), uintptr(w), uintptr(h),
	)
	return windows.Handle(r1)
}

func CreateRectRgn(x1, y1, x2, y2 int32) windows.Handle {
	r1, _, _ := procCreateRectRgn.Call(
		uintptr(x1), uintptr(y1), uintptr(x2), uintptr(y2),
	)
	return windows.Handle(r1)
}

func SelectClipRgn(hdc windows.Handle, hrgn windows.Handle) int32 {
	r1, _, _ := procSelectClipRgn.Call(uintptr(hdc), uintptr(hrgn))
	return int32(r1)
}

func SetTimer(hwnd windows.HWND, nIDEvent uintptr, uElapse uint32, lpTimerFunc uintptr) uintptr {
	r1, _, _ := procSetTimer.Call(uintptr(hwnd), nIDEvent, uintptr(uElapse), lpTimerFunc)
	return r1
}

func KillTimer(hwnd windows.HWND, uIDEvent uintptr) bool {
	r1, _, _ := procKillTimer.Call(uintptr(hwnd), uIDEvent)
	return r1 != 0
}

func FillRgn(hdc windows.Handle, hrgn windows.Handle, hbr windows.Handle) bool {

	r1, _, _ := procFillRgn.Call(uintptr(hdc), uintptr(hrgn), uintptr(hbr))
	return r1 != 0
}

func FrameRgn(hdc windows.Handle, hrgn windows.Handle, hbr windows.Handle, w, h int32) bool {
	r1, _, _ := procFrameRgn.Call(uintptr(hdc), uintptr(hrgn), uintptr(hbr), uintptr(w), uintptr(h))
	return r1 != 0
}

// Menu

func CreatePopupMenu() windows.Handle {
	r1, _, _ := procCreatePopupMenu.Call()
	return windows.Handle(r1)
}

func AppendMenu(hMenu windows.Handle, uFlags uint32, uIDNewItem uintptr, lpNewItem string) bool {
	var pText *uint16
	if lpNewItem != "" {
		pText, _ = windows.UTF16PtrFromString(lpNewItem)
	}
	r1, _, _ := procAppendMenuW.Call(
		uintptr(hMenu),
		uintptr(uFlags),
		uIDNewItem,
		uintptr(unsafe.Pointer(pText)),
	)
	return r1 != 0
}

func TrackPopupMenu(hMenu windows.Handle, uFlags uint32, x, y int32, nReserved int32, hWnd windows.HWND, prcRect *RECT) uint32 {
	r1, _, _ := procTrackPopupMenu.Call(
		uintptr(hMenu),
		uintptr(uFlags),
		uintptr(x),
		uintptr(y),
		uintptr(nReserved),
		uintptr(hWnd),
		uintptr(unsafe.Pointer(prcRect)),
	)
	return uint32(r1)
}

func DestroyMenu(hMenu windows.Handle) bool {
	r1, _, _ := procDestroyMenu.Call(uintptr(hMenu))
	return r1 != 0
}

// DPI & Metrics

// EnableDpiAwareness opts the process into Per-Monitor DPI Awareness V2
func EnableDpiAwareness() {
	procSetProcessDpiAwarenessContext := user32.NewProc("SetProcessDpiAwarenessContext")
	if procSetProcessDpiAwarenessContext.Find() == nil {
		// DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 = -4
		const DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 = ^uintptr(3)
		r1, _, _ := procSetProcessDpiAwarenessContext.Call(DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2)
		if r1 != 0 {
			return
		}
	}
	shcore := windows.NewLazyDLL("shcore.dll")
	if procSetProcessDpiAwareness := shcore.NewProc("SetProcessDpiAwareness"); procSetProcessDpiAwareness.Find() == nil {
		// PROCESS_PER_MONITOR_DPI_AWARE = 2
		r1, _, _ := procSetProcessDpiAwareness.Call(2)
		if r1 == 0 {
			return
		}
	}
	procSetProcessDPIAware := user32.NewProc("SetProcessDPIAware")
	if procSetProcessDPIAware.Find() == nil {
		_, _, _ = procSetProcessDPIAware.Call()
	}
}

func GetDpiForHwnd(hwnd windows.HWND) uint32 {
	if procGetDpiForWindow.Find() == nil {
		r1, _, _ := procGetDpiForWindow.Call(uintptr(hwnd))
		if r1 > 0 {
			return uint32(r1)
		}
	}
	return 96 // Default DPI (100%)
}

func ScaleDpi(val int32, dpi uint32) int32 {
	return (val * int32(dpi)) / 96
}

func GetSystemMetrics(nIndex int32) int32 {
	r1, _, _ := procGetSystemMetrics.Call(uintptr(nIndex))
	return int32(r1)
}

// UxTheme — Dark Mode & Visual Styles

// SetWindowTheme applies a named visual style to a window.
// Pass "DarkMode_Explorer" / "DarkMode_CFD" for native dark menus/controls.
func SetWindowTheme(hwnd windows.HWND, subAppName, subIdList *uint16) error {
	if uxtheme.Load() != nil {
		return nil
	}
	r1, _, err := procSetWindowTheme.Call(
		uintptr(hwnd),
		uintptr(unsafe.Pointer(subAppName)),
		uintptr(unsafe.Pointer(subIdList)),
	)
	if r1 != 0 {
		return err
	}
	return nil
}

// AllowDarkModeForWindow opts a specific HWND into the dark-mode rendering path.
// Must be called before the window is shown / before TrackPopupMenu.
func AllowDarkModeForWindow(hwnd windows.HWND, allow bool) {
	if err := uxtheme.Load(); err != nil {
		return
	}
	h := windows.Handle(uxtheme.Handle())
	// Try ordinal 133 (AllowDarkModeForWindow), fallback to named proc
	addr, err := windows.GetProcAddressByOrdinal(h, 133)
	if err != nil {
		addr, err = windows.GetProcAddress(h, "AllowDarkModeForWindow")
		if err != nil {
			return
		}
	}
	var v uintptr
	if allow {
		v = 1
	}
	syscall.SyscallN(addr, uintptr(hwnd), v)
}

// SetPreferredAppMode tells the Win10/11 dark-mode subsystem the app's preferred
// colour mode. mode=1 = AllowDark (auto), mode=2 = ForceDark, mode=0 = Default.
func SetPreferredAppMode(mode int32) {
	if err := uxtheme.Load(); err != nil {
		return
	}
	h := windows.Handle(uxtheme.Handle())
	// Try ordinal 135 (SetPreferredAppMode), fallback to ordinal 135 on 1809 (AllowDarkModeForApp)
	addr, err := windows.GetProcAddressByOrdinal(h, 135)
	if err != nil {
		addr, err = windows.GetProcAddress(h, "SetPreferredAppMode")
		if err != nil {
			return
		}
	}
	syscall.SyscallN(addr, uintptr(mode))
}

// DWM & Windows 11 Styling

func DwmSetWindowAttribute(hwnd windows.HWND, attribute uint32, value unsafe.Pointer, size uint32) uintptr {
	if dwmapi.Load() != nil {
		return 1
	}
	r1, _, _ := procDwmSetWindowAttribute.Call(
		uintptr(hwnd),
		uintptr(attribute),
		uintptr(value),
		uintptr(size),
	)
	return r1
}

func DwmGetColorizationColor(color *uint32, opaqueBlend *int32) uintptr {
	if dwmapi.Load() != nil {
		return 1
	}
	r1, _, _ := procDwmGetColorizationColor.Call(
		uintptr(unsafe.Pointer(color)),
		uintptr(unsafe.Pointer(opaqueBlend)),
	)
	return r1
}

// GetAppsUseLightTheme returns whether the user's app UI should use light mode.
// Reads AppsUseLightTheme first (controls app chrome colors), falls back to
// SystemUsesLightTheme (controls taskbar/tray). This ensures text colors and
// the acrylic backdrop tint are always computed from the same theme signal.
func GetAppsUseLightTheme() bool {
	k, err := registry.OpenKey(
		registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`,
		registry.QUERY_VALUE,
	)
	if err != nil {
		return false
	}
	defer k.Close()

	if val, _, err := k.GetIntegerValue("AppsUseLightTheme"); err == nil {
		return val != 0
	}
	if val, _, err := k.GetIntegerValue("SystemUsesLightTheme"); err == nil {
		return val != 0
	}
	return false
}

// GetLiveAccentColor retrieves the user's manual Windows accent color from registry,
// falling back to DWM or Windows 11 default blue (#0F6CBD).
func GetLiveAccentColor() (r, g, b uint8) {
	// 1. Check HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer\Accent -> AccentColorMenu
	k, err := registry.OpenKey(
		registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Explorer\Accent`,
		registry.QUERY_VALUE,
	)
	if err == nil {
		val, _, err := k.GetIntegerValue("AccentColorMenu")
		k.Close()
		if err == nil && val != 0 {
			// Format is 0xAABBGGRR
			return uint8(val & 0xFF), uint8((val >> 8) & 0xFF), uint8((val >> 16) & 0xFF)
		}
	}

	// 2. Check HKCU\Software\Microsoft\Windows\DWM -> AccentColor
	kDwm, err := registry.OpenKey(
		registry.CURRENT_USER,
		`Software\Microsoft\Windows\DWM`,
		registry.QUERY_VALUE,
	)
	if err == nil {
		val, _, err := kDwm.GetIntegerValue("AccentColor")
		kDwm.Close()
		if err == nil && val != 0 {
			// Format is 0xAABBGGRR
			return uint8(val & 0xFF), uint8((val >> 8) & 0xFF), uint8((val >> 16) & 0xFF)
		}
	}

	// 3. Fallback to DwmGetColorizationColor
	var rawColor uint32
	var opaque int32
	res := DwmGetColorizationColor(&rawColor, &opaque)
	if res == 0 && rawColor != 0 {
		// Format: 0xAARRGGBB
		red := uint8((rawColor >> 16) & 0xFF)
		green := uint8((rawColor >> 8) & 0xFF)
		blue := uint8(rawColor & 0xFF)
		return red, green, blue
	}

	// 4. Fallback: Windows 11 default blue (#0F6CBD)
	return 0x0F, 0x6C, 0xBD
}

// WindowsBuildNumber returns the Windows build number via RtlGetVersion (ntdll).
// This bypasses the compatibility shim that makes VerifyVersionInfo lie.
func WindowsBuildNumber() uint32 {
	type rtlOSVersionInfoEx struct {
		DwOSVersionInfoSize uint32
		DwMajorVersion      uint32
		DwMinorVersion      uint32
		DwBuildNumber       uint32
		DwPlatformId        uint32
		SzCSDVersion        [128]uint16
		WServicePackMajor   uint16
		WServicePackMinor   uint16
		WSuiteMask          uint16
		WProductType        uint8
		WReserved           uint8
	}
	var info rtlOSVersionInfoEx
	info.DwOSVersionInfoSize = uint32(unsafe.Sizeof(info))
	procRtlGetVersion.Call(uintptr(unsafe.Pointer(&info)))
	return info.DwBuildNumber
}

var (
	procSetWindowCompositionAttribute = user32.NewProc("SetWindowCompositionAttribute")
	procDwmExtendFrameIntoClientArea  = dwmapi.NewProc("DwmExtendFrameIntoClientArea")
)

type ACCENT_STATE int32

const (
	ACCENT_DISABLED                  ACCENT_STATE = 0
	ACCENT_ENABLE_GRADIENT            ACCENT_STATE = 1
	ACCENT_ENABLE_TRANSPARENTGRADIENT ACCENT_STATE = 2
	ACCENT_ENABLE_BLURBEHIND          ACCENT_STATE = 3
	ACCENT_ENABLE_ACRYLICBLURBEHIND   ACCENT_STATE = 4
	ACCENT_ENABLE_HOSTBACKDROP        ACCENT_STATE = 5
)

type ACCENT_POLICY struct {
	AccentState   ACCENT_STATE
	AccentFlags   uint32
	GradientColor uint32
	AnimationId   uint32
}

type WINDOWCOMPOSITIONATTRIBDATA struct {
	Attrib uint32
	PvData unsafe.Pointer
	CbData uint32
}

func EnableAcrylicBlur(hwnd windows.HWND, isDark bool) {
	type MARGINS struct {
		CxLeftWidth    int32
		CxRightWidth   int32
		CyTopHeight    int32
		CyBottomHeight int32
	}
	margins := MARGINS{-1, -1, -1, -1}
	_, _, _ = procDwmExtendFrameIntoClientArea.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&margins)))

	if procSetWindowCompositionAttribute.Find() == nil {
		var gradientColor uint32
		if isDark {
			gradientColor = 0x45181818 // ~27% translucent dark tint with clear frosted glass blur & noise
		} else {
			gradientColor = 0x12F8F8F8 // ~7% crystal clear translucent frosted glass blur & noise
		}

		policy := ACCENT_POLICY{
			AccentState:   ACCENT_ENABLE_ACRYLICBLURBEHIND,
			AccentFlags:   2,
			GradientColor: gradientColor,
		}

		data := WINDOWCOMPOSITIONATTRIBDATA{
			Attrib: 19, // WCA_ACCENT_POLICY
			PvData: unsafe.Pointer(&policy),
			CbData: uint32(unsafe.Sizeof(policy)),
		}

		_, _, _ = procSetWindowCompositionAttribute.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&data)))
	}
}

// ApplyWindows11Styling applies dark mode, rounded corners, Mica/Acrylic frosted glass backdrop.
func ApplyWindows11Styling(hwnd windows.HWND, isDark bool) {
	// 1. Dark / Light mode (Build 22000+)
	var darkModeVal int32
	if isDark {
		darkModeVal = 1
	}
	DwmSetWindowAttribute(
		hwnd,
		DWMWA_USE_IMMERSIVE_DARK_MODE,
		unsafe.Pointer(&darkModeVal),
		uint32(unsafe.Sizeof(darkModeVal)),
	)

	// 2. Rounded corners (Build 22000+)
	var cornerVal int32 = DWMWCP_ROUND
	DwmSetWindowAttribute(
		hwnd,
		DWMWA_WINDOW_CORNER_PREFERENCE,
		unsafe.Pointer(&cornerVal),
		uint32(unsafe.Sizeof(cornerVal)),
	)

	// 3. Acrylic Frosted Glass Blur with subtle noise
	EnableAcrylicBlur(hwnd, isDark)
}

// SetInterfaceDnsSettings calls iphlpapi.dll SetInterfaceDnsSettings
func SetInterfaceDnsSettings(guid *windows.GUID, settings *DNS_INTERFACE_SETTINGS) uint32 {
	if iphlpapi.Load() != nil {
		return uint32(windows.ERROR_NOT_SUPPORTED)
	}
	r1, _, _ := procSetInterfaceDnsSettings.Call(
		uintptr(unsafe.Pointer(guid)),
		uintptr(unsafe.Pointer(settings)),
	)
	return uint32(r1)
}

// GetInterfaceDnsSettings calls iphlpapi.dll GetInterfaceDnsSettings
func GetInterfaceDnsSettings(guid *windows.GUID, settings *DNS_INTERFACE_SETTINGS) uint32 {
	if iphlpapi.Load() != nil {
		return uint32(windows.ERROR_NOT_SUPPORTED)
	}
	r1, _, _ := procGetInterfaceDnsSettings.Call(
		uintptr(unsafe.Pointer(guid)),
		uintptr(unsafe.Pointer(settings)),
	)
	return uint32(r1)
}

// DnsFlushResolverCache flushes the Windows DNS cache
func DnsFlushResolverCache() bool {
	if dnsapi.Load() != nil {
		return false
	}
	r1, _, _ := procDnsFlushResolverCache.Call()
	return r1 != 0
}
