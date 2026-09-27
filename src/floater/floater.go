package floater

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"nodal/src/config"
	"nodal/src/dns"
	"nodal/src/elevation"
	"nodal/src/winapi"

	"golang.org/x/sys/windows"
)


const (
	className = "NodalFlyoutWindow"
)

type Floater struct {
	mu            sync.Mutex
	hwnd          windows.HWND
	visible       bool
	dpi           uint32
	isDark        bool
	accentR       uint8
	accentG       uint8
	accentB       uint8
	profiles      []config.DNSProfile
	activeName    string
	hoveredRow    int // -1 = none, 0..N = profiles, 98 = uac banner
	hoveredUAC    bool
	uacAlert      bool
	uacMessage    string
	adapterName   string
	adapterType   uint32
	privacyMode   string
	showCustomBtn bool
	marqueeOffset int32
	marqueePause  int32
	openTime      time.Time
	onProfileSet  func(profileName string, uacAlert bool)
	onConfigOpen  func()
}


func NewFloater(onProfileSet func(string, bool), onConfigOpen func()) (*Floater, error) {
	f := &Floater{
		dpi:           96,
		hoveredRow:    -1,
		activeName:    "DHCP",
		privacyMode:   config.PrivacyModeVisible,
		showCustomBtn: true,
		onProfileSet:  onProfileSet,
		onConfigOpen:  onConfigOpen,
	}

	if err := f.registerClass(); err != nil {
		return nil, err
	}

	if err := f.createWindow(); err != nil {
		return nil, err
	}

	f.updateTheme()
	return f, nil
}

func (f *Floater) registerClass() error {
	pClassName, _ := windows.UTF16PtrFromString(className)

	var wc winapi.WNDCLASSEXW
	wc.CbSize = uint32(unsafe.Sizeof(wc))
	wc.Style = 0
	wc.LpfnWndProc = syscall.NewCallback(floaterWndProc)
	wc.HInstance = 0
	wc.HCursor = windows.Handle(winapi.LoadCursor(32512))
	wc.LpszClassName = pClassName

	_, err := winapi.RegisterClassEx(&wc)
	if err != nil && !errors.Is(err, windows.ERROR_CLASS_ALREADY_EXISTS) {
		return fmt.Errorf("failed to register floater window class: %w", err)
	}
	return nil
}

var globalFloater *Floater

func floaterWndProc(hwnd windows.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	if globalFloater != nil && globalFloater.hwnd == hwnd {
		return globalFloater.handleMessage(hwnd, msg, wParam, lParam)
	}
	return winapi.DefWindowProc(hwnd, msg, wParam, lParam)
}

func (f *Floater) createWindow() error {
	pClassName, _ := windows.UTF16PtrFromString(className)
	pWindowName, _ := windows.UTF16PtrFromString("Nodal Flyout")

	hwnd, err := winapi.CreateWindowEx(
		winapi.WS_EX_TOPMOST|winapi.WS_EX_TOOLWINDOW,
		pClassName,
		pWindowName,
		winapi.WS_POPUP|winapi.WS_CLIPSIBLINGS,
		0, 0, 100, 100,
		0, 0, 0, 0,
	)
	if err != nil {
		return fmt.Errorf("failed to create floater window: %w", err)
	}

	f.hwnd = hwnd
	globalFloater = f
	f.dpi = winapi.GetDpiForHwnd(hwnd)
	return nil
}

func (f *Floater) updateTheme() {
	f.isDark = !winapi.GetAppsUseLightTheme()
	f.accentR, f.accentG, f.accentB = winapi.GetLiveAccentColor()
	winapi.ApplyWindows11Styling(f.hwnd, f.isDark)
}

// Toggle shows or hides the floater window near the tray icon
func (f *Floater) Toggle() {
	if f.visible {
		if time.Since(f.openTime) < 350*time.Millisecond {
			return
		}
		f.Hide()
	} else {
		f.Show()
	}
}

// Show opens the floater window and refreshes profiles from config
func (f *Floater) Show() {
	if f.visible {
		return
	}
	f.showInternal()
}

// Hide closes the floater window
func (f *Floater) Hide() {
	if !f.visible {
		return
	}
	f.visible = false
	winapi.KillTimer(f.hwnd, 1001)
	f.marqueeOffset = 0
	winapi.ShowWindow(f.hwnd, winapi.SW_HIDE)
}

func (f *Floater) showInternal() {
	// Reload config each time floater opens per spec §6
	cfg, err := config.LoadConfig()
	if err == nil {
		f.profiles = cfg.Profiles
		f.privacyMode = cfg.EffectivePrivacyMode()
		f.showCustomBtn = cfg.EffectiveShowCustomButton()
	}

	// Detect active adapter and current DNS
	if adapter, err := dns.GetActiveAdapter(); err == nil {
		f.adapterName = adapter.DisplayName()
		f.adapterType = adapter.IfType
		f.activeName = dns.DetectCurrentProfile(f.profiles)
	} else {
		f.adapterName = "Disconnected"
		f.adapterType = 0
		f.activeName = "DHCP"
	}

	// Reset marquee scroll
	f.marqueeOffset = 0
	f.marqueePause = 25

	// Update theme in case Windows theme changed
	f.updateTheme()

	// Calculate size and position
	f.repositionWindow()

	f.openTime = time.Now()
	f.visible = true
	winapi.ShowWindow(f.hwnd, winapi.SW_SHOW)
	winapi.SetForegroundWindow(f.hwnd)
	winapi.SetTimer(f.hwnd, 1001, 35, 0)
	winapi.InvalidateRect(f.hwnd, nil, true)
}

func (f *Floater) calculateDimensions() (w, h int32) {
	scale := func(v int32) int32 {
		return winapi.ScaleDpi(v, f.dpi)
	}

	w = scale(280) // Flyout width

	headerH := scale(42)
	rowH := scale(48)
	if f.privacyMode == config.PrivacyModeHidden {
		rowH = scale(38)
	}

	bannerH := int32(0)
	if f.uacAlert {
		bannerH = scale(36)
	}
	bottomPad := scale(12)

	rowsTotalH := int32(0)
	if len(f.profiles) == 0 {
		rowsTotalH = scale(96)
	} else {
		rowsTotalH = int32(len(f.profiles)) * rowH
		if len(f.profiles) < 5 && f.showCustomBtn {
			rowsTotalH += scale(32)
		}
	}

	h = headerH + rowsTotalH + bannerH + bottomPad
	return w, h
}

// Reposition floater adjacent to tray icon based on active monitor & taskbar edge
func (f *Floater) repositionWindow() {
	// Update DPI for this monitor/window first
	f.dpi = winapi.GetDpiForHwnd(f.hwnd)
	w, h := f.calculateDimensions()

	var anchorPt winapi.POINT
	var iconRect winapi.RECT
	hasIconRect := false

	// Query the exact screen coordinates of the tray icon so moving the mouse quickly won't throw off positioning.
	if rc, ok := winapi.GetTrayIconRect(0); ok {
		iconRect = rc
		hasIconRect = true
		anchorPt = winapi.POINT{
			X: rc.Left + rc.Width()/2,
			Y: rc.Top + rc.Height()/2,
		}
	} else {
		winapi.GetCursorPos(&anchorPt)
	}

	// Query the exact monitor containing the tray icon / anchor point
	hMon := winapi.MonitorFromPoint(anchorPt, winapi.MONITOR_DEFAULTTONEAREST)
	var mi winapi.MONITORINFO
	winapi.GetMonitorInfo(hMon, &mi)
	workArea := mi.RcWork
	monArea := mi.RcMonitor

	margin := winapi.ScaleDpi(4, f.dpi)

	// Query main taskbar window directly
	hTaskbar := winapi.FindWindow("Shell_TrayWnd", "")
	var rcTaskbar winapi.RECT
	hasTaskbar := hTaskbar != 0 && winapi.GetWindowRect(hTaskbar, &rcTaskbar)

	var posX, posY int32

	if hasIconRect {
		if hasTaskbar && rcTaskbar.Bottom > monArea.Top && rcTaskbar.Top < monArea.Bottom {
			if rcTaskbar.Top > monArea.Top+monArea.Height()/2 {
				// Taskbar is at the BOTTOM
				posX = anchorPt.X - w/2
				posY = iconRect.Top - h - margin
			} else if rcTaskbar.Bottom <= monArea.Top+monArea.Height()/2 {
				// Taskbar is at the TOP
				posX = anchorPt.X - w/2
				posY = iconRect.Bottom + margin
			} else if rcTaskbar.Left > monArea.Left+monArea.Width()/2 {
				// Taskbar is at the RIGHT
				posX = iconRect.Left - w - margin
				posY = anchorPt.Y - h/2
			} else {
				// Taskbar is at the LEFT
				posX = iconRect.Right + margin
				posY = anchorPt.Y - h/2
			}
		} else {
			posX = anchorPt.X - w/2
			posY = iconRect.Top - h - margin
		}
	} else if hasTaskbar && rcTaskbar.Bottom > monArea.Top && rcTaskbar.Top < monArea.Bottom {
		if rcTaskbar.Top > monArea.Top+monArea.Height()/2 {
			// Taskbar is at the BOTTOM
			posX = anchorPt.X - w/2
			posY = rcTaskbar.Top - h - margin
		} else if rcTaskbar.Bottom <= monArea.Top+monArea.Height()/2 {
			// Taskbar is at the TOP
			posX = anchorPt.X - w/2
			posY = rcTaskbar.Bottom + margin
		} else if rcTaskbar.Left > monArea.Left+monArea.Width()/2 {
			// Taskbar is at the RIGHT
			posX = rcTaskbar.Left - w - margin
			posY = anchorPt.Y - h/2
		} else {
			// Taskbar is at the LEFT
			posX = rcTaskbar.Right + margin
			posY = anchorPt.Y - h/2
		}
	} else if workArea.Bottom < monArea.Bottom {
		posX = anchorPt.X - w/2
		posY = workArea.Bottom - h - margin
	} else if workArea.Top > monArea.Top {
		posX = anchorPt.X - w/2
		posY = workArea.Top + margin
	} else if workArea.Left > monArea.Left {
		posX = workArea.Left + margin
		posY = anchorPt.Y - h/2
	} else if workArea.Right < monArea.Right {
		posX = workArea.Right - w - margin
		posY = anchorPt.Y - h/2
	} else {
		posX = anchorPt.X - w/2
		posY = monArea.Bottom - h - margin
	}

	// Strict clamping within this monitor's working area
	if posX+w > workArea.Right-margin {
		posX = workArea.Right - w - margin
	}
	if posX < workArea.Left+margin {
		posX = workArea.Left + margin
	}
	if posY+h > workArea.Bottom-margin {
		posY = workArea.Bottom - h - margin
	}
	if posY < workArea.Top+margin {
		posY = workArea.Top + margin
	}

	winapi.SetWindowPos(f.hwnd, winapi.HWND_TOPMOST, posX, posY, w, h, winapi.SWP_SHOWWINDOW)
}

func (f *Floater) handleMessage(hwnd windows.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case winapi.WM_TIMER:
		if wParam == 1001 && f.visible {
			if f.marqueePause > 0 {
				f.marqueePause--
			} else {
				f.marqueeOffset++
			}
			w, _ := f.calculateDimensions()
			headerRect := winapi.RECT{Left: 0, Top: 0, Right: w, Bottom: winapi.ScaleDpi(40, f.dpi)}
			winapi.InvalidateRect(f.hwnd, &headerRect, false)
		}
		return 0

	case winapi.WM_ACTIVATE:
		// Auto-dismiss on loss of focus matching native Windows 11 flyouts
		if (wParam & 0xFFFF) == winapi.WA_INACTIVE {
			// Post deferred hide to avoid re-entrancy inside wndproc
			winapi.PostMessage(hwnd, winapi.WM_USER+100, 0, 0)
		}
		return 0

	case winapi.WM_KILLFOCUS:
		winapi.PostMessage(hwnd, winapi.WM_USER+100, 0, 0)
		return 0

	case winapi.WM_USER + 100:
		f.Hide()
		return 0


	case winapi.WM_SETTINGCHANGE:
		f.updateTheme()
		winapi.InvalidateRect(f.hwnd, nil, true)
		return 0

	case winapi.WM_DPICHANGED:
		f.dpi = uint32(wParam & 0xFFFF)
		f.repositionWindow()
		winapi.InvalidateRect(f.hwnd, nil, true)
		return 0

	case winapi.WM_ERASEBKGND:
		return 1

	case winapi.WM_PAINT:
		var ps winapi.PAINTSTRUCT
		hdc := winapi.BeginPaint(hwnd, &ps)
		f.paint(hdc)
		winapi.EndPaint(hwnd, &ps)
		return 0

	case winapi.WM_MOUSEMOVE:
		pt := winapi.POINT{
			X: int32(int16(lParam & 0xFFFF)),
			Y: int32(int16((lParam >> 16) & 0xFFFF)),
		}
		f.handleMouseMove(pt)
		return 0

	case winapi.WM_MOUSELEAVE:
		f.hoveredRow = -1
		f.hoveredUAC = false
		winapi.InvalidateRect(f.hwnd, nil, false)
		return 0

	case winapi.WM_LBUTTONUP:
		f.handleClick()
		return 0
	}

	return winapi.DefWindowProc(hwnd, msg, wParam, lParam)
}

func (f *Floater) handleMouseMove(pt winapi.POINT) {
	var tme winapi.TRACKMOUSEEVENT
	tme.CbSize = uint32(unsafe.Sizeof(tme))
	tme.DwFlags = winapi.TME_LEAVE
	tme.HwndTrack = f.hwnd
	winapi.TrackMouse(&tme)

	var rcClient winapi.RECT
	winapi.GetClientRect(f.hwnd, &rcClient)
	w := rcClient.Width()
	if w <= 0 {
		w, _ = f.calculateDimensions()
	}
	scale := func(v int32) int32 { return winapi.ScaleDpi(v, f.dpi) }

	headerH := scale(42)
	rowH := scale(48)
	if f.privacyMode == config.PrivacyModeHidden {
		rowH = scale(38)
	}
	sidePad := scale(12)

	prevRow := f.hoveredRow
	prevUAC := f.hoveredUAC

	f.hoveredRow = -1
	f.hoveredUAC = false

	currY := headerH
	if len(f.profiles) == 0 {
		btnRect := winapi.RECT{
			Left:   sidePad + scale(10),
			Top:    currY + scale(38),
			Right:  w - sidePad - scale(10),
			Bottom: currY + scale(74),
		}
		if pt.X >= btnRect.Left && pt.X <= btnRect.Right && pt.Y >= btnRect.Top && pt.Y <= btnRect.Bottom {
			f.hoveredRow = 0
		}
		currY += scale(84)
	} else {
		for i := range f.profiles {
			rowRect := winapi.RECT{
				Left:   sidePad,
				Top:    currY,
				Right:  w - sidePad,
				Bottom: currY + rowH - scale(4),
			}
			if pt.X >= rowRect.Left && pt.X <= rowRect.Right && pt.Y >= rowRect.Top && pt.Y <= rowRect.Bottom {
				f.hoveredRow = i
			}
			currY += rowH
		}

		if len(f.profiles) < 5 && f.showCustomBtn {
			btnRect := winapi.RECT{
				Left:   sidePad,
				Top:    currY + scale(2),
				Right:  w - sidePad,
				Bottom: currY + scale(30),
			}
			if f.hoveredRow == -1 && pt.X >= btnRect.Left && pt.X <= btnRect.Right && pt.Y >= btnRect.Top && pt.Y <= btnRect.Bottom {
				f.hoveredRow = 99
			}
			currY += scale(32)
		}
	}

	// Check UAC banner
	if f.uacAlert {
		bannerH := scale(34)
		if pt.Y >= currY && pt.Y <= currY+bannerH {
			f.hoveredUAC = true
		}
		currY += bannerH
	}

	if prevRow != f.hoveredRow || prevUAC != f.hoveredUAC {
		winapi.InvalidateRect(f.hwnd, nil, false)
	}
}

func (f *Floater) handleClick() {
	row := f.hoveredRow
	uac := f.hoveredUAC
	f.hoveredRow = -1
	f.hoveredUAC = false

	if uac {
		// User clicked "Grant Admin Access"
		go func() {
			err := elevation.TriggerUACRegistration()
			f.mu.Lock()
			if err == nil {
				f.uacAlert = false
				f.uacMessage = ""
			} else {
				f.uacAlert = true
				f.uacMessage = "UAC permission declined"
			}
			f.mu.Unlock()
			winapi.InvalidateRect(f.hwnd, nil, true)
		}()
		return
	}

	if len(f.profiles) == 0 {
		if row == 0 {
			f.Hide()
			if f.onConfigOpen != nil {
				f.onConfigOpen()
			} else {
				_ = config.OpenInEditor()
			}
		}
		return
	}

	if row == 99 {
		f.Hide()
		if f.onConfigOpen != nil {
			f.onConfigOpen()
		} else {
			_ = config.OpenInEditor()
		}
		return
	}

	if row >= 0 && row < len(f.profiles) {
		selected := f.profiles[row]
		go func() {
			defer func() { recover() }()
			err := elevation.ApplyProfileElevated(selected)
			f.mu.Lock()
			if err != nil {
				f.uacAlert = true
				f.uacMessage = "Admin access required"
			} else {
				f.activeName = selected.Name
				f.uacAlert = false
				f.uacMessage = ""
			}
			active := f.activeName
			alert := f.uacAlert
			f.mu.Unlock()

			if f.onProfileSet != nil {
				f.onProfileSet(active, alert)
			}
			winapi.InvalidateRect(f.hwnd, nil, true)
		}()
	}
}

// GDI rendering of Windows 11 Fluent Flyout with double buffering for smooth DWM acrylic backdrop
func (f *Floater) paint(hdc windows.Handle) {
	var rcClient winapi.RECT
	winapi.GetClientRect(f.hwnd, &rcClient)
	w := rcClient.Width()
	h := rcClient.Height()
	if w <= 0 || h <= 0 {
		w, h = f.calculateDimensions()
	}

	memDC := winapi.CreateCompatibleDC(hdc)
	defer winapi.DeleteDC(memDC)

	memBmp := winapi.CreateCompatibleBitmap(hdc, w, h)
	defer winapi.DeleteObject(memBmp)

	oldBmp := winapi.SelectObject(memDC, memBmp)
	defer winapi.SelectObject(memDC, oldBmp)

	// In DWM Acrylic composition, painting pure black (0x000000) onto the 32-bit buffer
	// makes that region transparent and shows through the Acrylic frosted glass & noise backdrop.
	bgRect := winapi.RECT{Left: 0, Top: 0, Right: w, Bottom: h}
	hbrBg := winapi.CreateSolidBrush(0x000000)
	winapi.FillRect(memDC, &bgRect, hbrBg)
	winapi.DeleteObject(hbrBg)

	dc := memDC

	// Colors for Win11 Dark/Light theme
	var (
		borderCol uint32 = 0x383838
		textMain  uint32 = 0xFFFFFF
		textSub   uint32 = 0x8C8C8C
		hoverPill uint32 = 0x262626
		accentRGB uint32 = uint32(f.accentR) | (uint32(f.accentG) << 8) | (uint32(f.accentB) << 16)
	)

	if !f.isDark {
		borderCol = 0xE5E5E5
		textMain = 0x1A1A1A
		textSub = 0x666666
		hoverPill = 0xECECEC
	}

	// Single source of truth for accent blends
	blendChan := func(fg, bg uint8, alpha float64) uint8 {
		return uint8(float64(fg)*alpha + float64(bg)*(1-alpha))
	}
	var bgR, bgG, bgB uint8
	if f.isDark {
		bgR, bgG, bgB = 0x20, 0x20, 0x20
	} else {
		bgR, bgG, bgB = 0xF9, 0xF9, 0xF9
	}
	// Subtle 14% accent fill & delicate 32% border for active state
	activePillFill := uint32(blendChan(f.accentR, bgR, 0.14)) | (uint32(blendChan(f.accentG, bgG, 0.14)) << 8) | (uint32(blendChan(f.accentB, bgB, 0.14)) << 16)
	activePillBorder := uint32(blendChan(f.accentR, bgR, 0.32)) | (uint32(blendChan(f.accentG, bgG, 0.32)) << 8) | (uint32(blendChan(f.accentB, bgB, 0.32)) << 16)

	winapi.SetBkMode(dc, winapi.TRANSPARENT)

	scale := func(v int32) int32 { return winapi.ScaleDpi(v, f.dpi) }

	// Typography setup (ClearType Natural Antialiasing, Segoe UI Variable / Segoe UI)
	const cleartypeNatural = 6

	fontTitle := winapi.CreateFont(-scale(13), 0, 0, 0, 600, 0, 0, 0, 1, 0, 0, cleartypeNatural, 0, "Segoe UI Variable Text")
	defer winapi.DeleteObject(fontTitle)

	fontBold := winapi.CreateFont(-scale(12), 0, 0, 0, 600, 0, 0, 0, 1, 0, 0, cleartypeNatural, 0, "Segoe UI Variable Text")
	defer winapi.DeleteObject(fontBold)

	fontRegular := winapi.CreateFont(-scale(12), 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, cleartypeNatural, 0, "Segoe UI Variable Text")
	defer winapi.DeleteObject(fontRegular)

	fontSmall := winapi.CreateFont(-scale(10), 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, cleartypeNatural, 0, "Segoe UI Variable Text")
	defer winapi.DeleteObject(fontSmall)

	fontTag := winapi.CreateFont(-scale(9), 0, 0, 0, 600, 0, 0, 0, 1, 0, 0, cleartypeNatural, 0, "Segoe UI Variable Text")
	defer winapi.DeleteObject(fontTag)

	sidePad := scale(12)

	// 3. Header: "Nodal" on left, [Icon + Adapter Name / Marquee] on same row right-aligned
	oldFont := winapi.SelectObject(dc, fontTitle)
	winapi.SetTextColor(dc, textMain)

	titleRect := winapi.RECT{Left: sidePad + scale(4), Top: scale(10), Right: sidePad + scale(75), Bottom: scale(34)}
	winapi.DrawText(dc, "Nodal", &titleRect, 0x0024) // DT_VCENTER | DT_SINGLELINE

	// Adapter glyph (Wi-Fi, Ethernet, VPN, Disconnected)
	var adapterGlyph string
	switch {
	case f.adapterName == "Disconnected" || f.adapterName == "No active connection" || f.adapterType == 0:
		adapterGlyph = "\uEB55" // Disconnected
	case f.adapterType == 131 || strings.Contains(strings.ToLower(f.adapterName), "vpn") || strings.Contains(strings.ToLower(f.adapterName), "wireguard"):
		adapterGlyph = "\uE705" // VPN
	case f.adapterType == 71:
		adapterGlyph = "\uE701" // Wi-Fi
	default:
		adapterGlyph = "\uE839" // Ethernet
	}

	modeText := f.adapterName
	if modeText == "" {
		modeText = "Disconnected"
	}

	// Measure mode text width
	winapi.SelectObject(dc, fontSmall)
	calcRect := winapi.RECT{Left: 0, Top: 0, Right: w, Bottom: scale(34)}
	winapi.DrawText(dc, modeText, &calcRect, 0x0420) // DT_CALCRECT | DT_SINGLELINE
	textW := calcRect.Right - calcRect.Left

	glyphSize := scale(14)
	glyphGap := scale(8)
	availRightW := w - (sidePad + scale(4) + scale(75)) - (sidePad + scale(4))
	maxTextW := availRightW - glyphSize - glyphGap

	if textW <= maxTextW {
		// Fits without marquee
		totalRightW := glyphSize + glyphGap + textW
		startX := w - (sidePad + scale(4)) - totalRightW

		// Draw adapter icon glyph
		fontGlyph := winapi.CreateFont(-scale(13), 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, cleartypeNatural, 0, "Segoe Fluent Icons")
		winapi.SelectObject(dc, fontGlyph)
		winapi.SetTextColor(dc, textSub)
		glyphRect := winapi.RECT{Left: startX, Top: scale(10), Right: startX + glyphSize, Bottom: scale(34)}
		winapi.DrawText(dc, adapterGlyph, &glyphRect, 0x0025) // DT_CENTER | DT_VCENTER | DT_SINGLELINE
		winapi.DeleteObject(fontGlyph)

		// Draw adapter name
		winapi.SelectObject(dc, fontSmall)
		winapi.SetTextColor(dc, textSub)
		nameRect := winapi.RECT{Left: startX + glyphSize + glyphGap, Top: scale(10), Right: startX + glyphSize + glyphGap + textW, Bottom: scale(34)}
		winapi.DrawText(dc, modeText, &nameRect, 0x0024) // DT_VCENTER | DT_SINGLELINE
	} else {
		// Overflows -> Smooth infinite marquee ticker
		startX := w - (sidePad + scale(4)) - availRightW

		// Draw adapter icon glyph
		fontGlyph := winapi.CreateFont(-scale(13), 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, cleartypeNatural, 0, "Segoe Fluent Icons")
		winapi.SelectObject(dc, fontGlyph)
		winapi.SetTextColor(dc, textSub)
		glyphRect := winapi.RECT{Left: startX, Top: scale(10), Right: startX + glyphSize, Bottom: scale(34)}
		winapi.DrawText(dc, adapterGlyph, &glyphRect, 0x0025) // DT_CENTER | DT_VCENTER | DT_SINGLELINE
		winapi.DeleteObject(fontGlyph)

		// Viewport for marquee text
		nameRect := winapi.RECT{Left: startX + glyphSize + glyphGap, Top: scale(10), Right: w - (sidePad + scale(4)), Bottom: scale(34)}

		winapi.SelectObject(dc, fontSmall)
		winapi.SetTextColor(dc, textSub)

		gap := scale(28)
		cycle := textW + gap
		if f.marqueeOffset >= cycle {
			f.marqueeOffset = 0
			f.marqueePause = 25
		}

		// Clip strictly within nameRect
		hrgnClip := winapi.CreateRectRgn(nameRect.Left, nameRect.Top, nameRect.Right, nameRect.Bottom)
		winapi.SelectClipRgn(dc, hrgnClip)

		// Draw primary instance
		drawX1 := nameRect.Left - f.marqueeOffset
		drawRect1 := winapi.RECT{Left: drawX1, Top: nameRect.Top, Right: drawX1 + textW, Bottom: nameRect.Bottom}
		winapi.DrawText(dc, modeText, &drawRect1, 0x0020) // DT_VCENTER | DT_SINGLELINE

		// Draw looping secondary instance
		drawX2 := drawX1 + cycle
		drawRect2 := winapi.RECT{Left: drawX2, Top: nameRect.Top, Right: drawX2 + textW, Bottom: nameRect.Bottom}
		winapi.DrawText(dc, modeText, &drawRect2, 0x0020) // DT_VCENTER | DT_SINGLELINE

		// Restore clip
		winapi.SelectClipRgn(dc, 0)
		winapi.DeleteObject(hrgnClip)
	}

	// 4. DNS Profile Rows or Empty State
	currY := scale(42)
	rowH := scale(48)
	if f.privacyMode == config.PrivacyModeHidden {
		rowH = scale(38)
	}

	if len(f.profiles) == 0 {
		// Empty State
		infoRect := winapi.RECT{
			Left:   sidePad,
			Top:    currY + scale(8),
			Right:  w - sidePad,
			Bottom: currY + scale(28),
		}
		winapi.SelectObject(dc, fontSmall)
		winapi.SetTextColor(dc, textSub)
		winapi.DrawText(dc, "No DNS presets in config.toml", &infoRect, 0x0001|0x0020) // DT_CENTER | DT_VCENTER

		btnRect := winapi.RECT{
			Left:   sidePad + scale(10),
			Top:    currY + scale(38),
			Right:  w - sidePad - scale(10),
			Bottom: currY + scale(74),
		}
		isHovered := f.hoveredRow == 0
		rgn := winapi.CreateRoundRectRgn(btnRect.Left, btnRect.Top, btnRect.Right, btnRect.Bottom, scale(6), scale(6))
		if isHovered {
			hbr := winapi.CreateSolidBrush(hoverPill)
			winapi.FillRgn(dc, rgn, hbr)
			winapi.DeleteObject(hbr)
		}
		hbrBorder := winapi.CreateSolidBrush(borderCol)
		winapi.FrameRgn(dc, rgn, hbrBorder, 1, 1)
		winapi.DeleteObject(hbrBorder)
		winapi.DeleteObject(rgn)

		winapi.SelectObject(dc, fontBold)
		winapi.SetTextColor(dc, accentRGB)
		winapi.DrawText(dc, "+ Add DNS Provider", &btnRect, 0x0001|0x0024) // DT_CENTER | DT_VCENTER | DT_SINGLELINE

		currY += scale(84)
	} else {
		for i, p := range f.profiles {
			isActive := p.Name == f.activeName
			isHovered := f.hoveredRow == i

			rowRect := winapi.RECT{
				Left:   sidePad,
				Top:    currY,
				Right:  w - sidePad,
				Bottom: currY + rowH - scale(4),
			}

			// Row Pill Background
			if isActive {
				rgn := winapi.CreateRoundRectRgn(rowRect.Left, rowRect.Top, rowRect.Right, rowRect.Bottom, scale(6), scale(6))
				hbrActiveFill := winapi.CreateSolidBrush(activePillFill)
				winapi.FillRgn(dc, rgn, hbrActiveFill)
				winapi.DeleteObject(hbrActiveFill)

				// Delicate, subtle accent border
				hbrActiveBorder := winapi.CreateSolidBrush(activePillBorder)
				winapi.FrameRgn(dc, rgn, hbrActiveBorder, 1, 1)
				winapi.DeleteObject(hbrActiveBorder)
				winapi.DeleteObject(rgn)

				// Windows 11 left vertical pill accent bar
				barW := scale(3)
				barH := scale(16)
				barX := rowRect.Left + scale(4)
				barY := rowRect.Top + (rowRect.Height()-barH)/2
				barRgn := winapi.CreateRoundRectRgn(barX, barY, barX+barW, barY+barH, scale(2), scale(2))
				hbrDot := winapi.CreateSolidBrush(accentRGB)
				winapi.FillRgn(dc, barRgn, hbrDot)
				winapi.DeleteObject(barRgn)
				winapi.DeleteObject(hbrDot)
			} else if isHovered {
				rgn := winapi.CreateRoundRectRgn(rowRect.Left, rowRect.Top, rowRect.Right, rowRect.Bottom, scale(6), scale(6))
				hbrHov := winapi.CreateSolidBrush(hoverPill)
				winapi.FillRgn(dc, rgn, hbrHov)
				winapi.DeleteObject(hbrHov)
				winapi.DeleteObject(rgn)
			}

			// Right-aligned Tag Badge (Clean, minimal, non-distracting chip)
			maxTextRight := rowRect.Right - scale(12)
			if p.Tag != "" {
				winapi.SelectObject(dc, fontTag)
				tagCalcRect := winapi.RECT{Left: 0, Top: 0, Right: w, Bottom: scale(20)}
				winapi.DrawText(dc, p.Tag, &tagCalcRect, 0x0420) // DT_CALCRECT | DT_SINGLELINE
				tagTextW := tagCalcRect.Right - tagCalcRect.Left

				badgeH := scale(18)
				badgeW := tagTextW + scale(12)
				badgeRight := rowRect.Right - scale(10)
				badgeLeft := badgeRight - badgeW
				badgeTop := rowRect.Top + (rowRect.Height()-badgeH)/2
				badgeBottom := badgeTop + badgeH
				badgeRect := winapi.RECT{Left: badgeLeft, Top: badgeTop, Right: badgeRight, Bottom: badgeBottom}

				badgeRgn := winapi.CreateRoundRectRgn(badgeLeft, badgeTop, badgeRight, badgeBottom, scale(4), scale(4))

				var (
					tagBgCol     uint32 = 0x222222
					tagBorderCol uint32 = 0x363636
					tagTextCol   uint32 = 0x888888
				)
				if !f.isDark {
					tagBgCol = 0xEEEEEE
					tagBorderCol = 0xD8D8D8
					tagTextCol = 0x666666
				}

				if isActive {
					tagBgCol = activePillFill
					tagBorderCol = activePillBorder
					tagTextCol = accentRGB
				} else if isHovered {
					tagBgCol = 0x2C2C2C
					tagBorderCol = 0x444444
					tagTextCol = 0xC0C0C0
					if !f.isDark {
						tagBgCol = 0xE4E4E4
						tagBorderCol = 0xCCCCCC
						tagTextCol = 0x333333
					}
				}

				hbrTagBg := winapi.CreateSolidBrush(tagBgCol)
				winapi.FillRgn(dc, badgeRgn, hbrTagBg)
				winapi.DeleteObject(hbrTagBg)

				hbrTagBorder := winapi.CreateSolidBrush(tagBorderCol)
				winapi.FrameRgn(dc, badgeRgn, hbrTagBorder, 1, 1)
				winapi.DeleteObject(hbrTagBorder)
				winapi.DeleteObject(badgeRgn)

				winapi.SelectObject(dc, fontTag)
				winapi.SetTextColor(dc, tagTextCol)
				winapi.DrawText(dc, p.Tag, &badgeRect, 0x0001|0x0024) // DT_CENTER | DT_VCENTER | DT_SINGLELINE

				maxTextRight = badgeLeft - scale(8)
			}

			// Text: Profile Name
			textX := rowRect.Left + scale(15)
			if isActive {
				winapi.SelectObject(dc, fontBold)
				winapi.SetTextColor(dc, textMain)
			} else {
				winapi.SelectObject(dc, fontRegular)
				winapi.SetTextColor(dc, textMain)
			}

			if f.privacyMode == config.PrivacyModeHidden {
				// Clean single-line vertically centered profile name
				nameRect := winapi.RECT{
					Left:   textX,
					Top:    rowRect.Top,
					Right:  maxTextRight,
					Bottom: rowRect.Bottom,
				}
				winapi.DrawText(dc, p.Name, &nameRect, 0x0024) // DT_VCENTER | DT_SINGLELINE
			} else {
				nameRect := winapi.RECT{
					Left:   textX,
					Top:    rowRect.Top + scale(6),
					Right:  maxTextRight,
					Bottom: rowRect.Top + scale(24),
				}
				winapi.DrawText(dc, p.Name, &nameRect, 0x0000)

				// Secondary text: IPs or "Automatic DNS"
				winapi.SelectObject(dc, fontSmall)
				winapi.SetTextColor(dc, textSub)

				var ipLabel string
				if p.Primary != "" {
					prim := p.Primary
					sec := p.Secondary
					if f.privacyMode == config.PrivacyModeMasked {
						prim = config.MaskIP(prim)
						if sec != "" {
							sec = config.MaskIP(sec)
						}
					}
					ipLabel = prim
					if sec != "" {
						ipLabel += " • " + sec
					}
				} else {
					ipLabel = "Automatic DNS"
				}

				ipRect := winapi.RECT{
					Left:   textX,
					Top:    rowRect.Top + scale(24),
					Right:  maxTextRight,
					Bottom: rowRect.Bottom - scale(4),
				}
				winapi.DrawText(dc, ipLabel, &ipRect, 0x0000)
			}

			currY += rowH
		}

		if len(f.profiles) < 5 && f.showCustomBtn {
			btnRect := winapi.RECT{
				Left:   sidePad,
				Top:    currY + scale(2),
				Right:  w - sidePad,
				Bottom: currY + scale(30),
			}
			isHovered := f.hoveredRow == 99
			rgn := winapi.CreateRoundRectRgn(btnRect.Left, btnRect.Top, btnRect.Right, btnRect.Bottom, scale(5), scale(5))
			if isHovered {
				hbr := winapi.CreateSolidBrush(hoverPill)
				winapi.FillRgn(dc, rgn, hbr)
				winapi.DeleteObject(hbr)
			}
			winapi.DeleteObject(rgn)

			winapi.SelectObject(dc, fontSmall)
			winapi.SetTextColor(dc, accentRGB)
			winapi.DrawText(dc, "+ Custom DNS", &btnRect, 0x0001|0x0024) // DT_CENTER | DT_VCENTER | DT_SINGLELINE

			currY += scale(32)
		}
	}

	// 5. UAC Warning InfoBar (filled, WinUI 3-style)
	if f.uacAlert {
		var infoBarBg uint32 = 0x18222A
		if !f.isDark {
			infoBarBg = 0xE2F3FE
		}
		const amberBGR uint32 = 0x48A9F7

		bannerH := scale(36)
		bannerRect := winapi.RECT{
			Left:   sidePad,
			Top:    currY + scale(2),
			Right:  w - sidePad,
			Bottom: currY + scale(2) + bannerH,
		}

		// Filled rounded background
		rgn := winapi.CreateRoundRectRgn(bannerRect.Left, bannerRect.Top, bannerRect.Right, bannerRect.Bottom, scale(6), scale(6))
		hbrBg := winapi.CreateSolidBrush(infoBarBg)
		winapi.FillRgn(dc, rgn, hbrBg)
		winapi.DeleteObject(hbrBg)
		winapi.DeleteObject(rgn)

		// 3px left accent stripe
		stripeRgn := winapi.CreateRoundRectRgn(bannerRect.Left, bannerRect.Top, bannerRect.Left+scale(3), bannerRect.Bottom, scale(3), scale(3))
		hbrStripe := winapi.CreateSolidBrush(amberBGR)
		winapi.FillRgn(dc, stripeRgn, hbrStripe)
		winapi.DeleteObject(hbrStripe)
		winapi.DeleteObject(stripeRgn)

		// Warning glyph E7BA (Segoe Fluent Icons)
		const warningGlyph = "\uE7BA"
		fontGlyph := winapi.CreateFont(-scale(14), 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, cleartypeNatural, 0, "Segoe Fluent Icons")
		winapi.SelectObject(dc, fontGlyph)
		winapi.SetTextColor(dc, amberBGR)
		glyphRect := winapi.RECT{
			Left:   bannerRect.Left + scale(10),
			Top:    bannerRect.Top,
			Right:  bannerRect.Left + scale(30),
			Bottom: bannerRect.Bottom,
		}
		winapi.DrawText(dc, warningGlyph, &glyphRect, 0x0024) // DT_VCENTER|DT_SINGLELINE
		winapi.DeleteObject(fontGlyph)

		// Banner message text
		txt := "Grant Admin Access"
		if f.uacMessage != "" {
			txt = f.uacMessage + " — click to fix"
		}
		winapi.SelectObject(dc, fontSmall)
		winapi.SetTextColor(dc, amberBGR)
		txtRect := winapi.RECT{
			Left:   bannerRect.Left + scale(30),
			Top:    bannerRect.Top,
			Right:  bannerRect.Right - scale(4),
			Bottom: bannerRect.Bottom,
		}
		winapi.DrawText(dc, txt, &txtRect, 0x0024)
	}

	winapi.SelectObject(dc, oldFont)
	winapi.BitBlt(hdc, 0, 0, w, h, memDC, 0, 0, winapi.SRCCOPY)
}
