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

type ThemePalette struct {
	TextMain         uint32
	TextSub          uint32
	Border           uint32
	HoverPill        uint32
	ActivePillFill   uint32
	ActivePillBorder uint32
	TagBg            uint32
	TagBorder        uint32
	TagText          uint32
	TagHoverBg       uint32
	TagHoverBorder   uint32
	TagHoverText     uint32
	InfoBarBg        uint32
	InfoBarText      uint32
}

func getThemePalette(isDark bool, accentR, accentG, accentB uint8) ThemePalette {
	blendChan := func(fg, bg uint8, alpha float64) uint8 {
		return uint8(float64(fg)*alpha + float64(bg)*(1-alpha))
	}

	if isDark {
		return ThemePalette{
			TextMain:         0x00FFFFFF,
			TextSub:          0x008C8C8C,
			Border:           0x00383838,
			HoverPill:        0x00262626,
			ActivePillFill:   uint32(blendChan(accentR, 0x20, 0.14)) | (uint32(blendChan(accentG, 0x20, 0.14)) << 8) | (uint32(blendChan(accentB, 0x20, 0.14)) << 16),
			ActivePillBorder: uint32(blendChan(accentR, 0x20, 0.32)) | (uint32(blendChan(accentG, 0x20, 0.32)) << 8) | (uint32(blendChan(accentB, 0x20, 0.32)) << 16),
			TagBg:            0x00222222,
			TagBorder:        0x00363636,
			TagText:          0x00888888,
			TagHoverBg:       0x002C2C2C,
			TagHoverBorder:   0x00444444,
			TagHoverText:     0x00C0C0C0,
			InfoBarBg:        0x0018222A,
			InfoBarText:      0x0048A9F7,
		}
	}

	return ThemePalette{
		TextMain:         0x001C1C1C, // Deep authoritative Windows 11 charcoal (#1C1C1C) — high contrast
		TextSub:          0x00525252, // Medium dark slate (#525252) — high contrast, never washed out
		Border:           0x00E0DEDC, // Crisp hairline divider/card border
		HoverPill:        0x00EFEFEF, // Clean, smooth hover pill
		ActivePillFill:   0x00FFFFFF, // Crisp white elevated card for selected DNS
		ActivePillBorder: 0x00DCDCDC, // Clean card outline
		TagBg:            0x00F3F2F1, // Clean airy badge pill
		TagBorder:        0x00DCDAD8, // Hairline badge border
		TagText:          0x00383634, // High-contrast readable badge text
		TagHoverBg:       0x00E6E5E4,
		TagHoverBorder:   0x00CAC8C6,
		TagHoverText:     0x001C1C1C,
		InfoBarBg:        0x00DCF0FF,
		InfoBarText:      0x00005FB8,
	}
}

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
	themeSetting  string
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
		themeSetting:  "auto",
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
	if f.themeSetting == "dark" {
		f.isDark = true
	} else if f.themeSetting == "light" {
		f.isDark = false
	} else {
		f.isDark = !winapi.GetAppsUseLightTheme()
	}
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
		f.themeSetting = cfg.EffectiveTheme()
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
			rowsTotalH += scale(38)
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
			btnH := scale(32)
			btnRect := winapi.RECT{
				Left:   sidePad,
				Top:    currY + scale(4),
				Right:  w - sidePad,
				Bottom: currY + scale(4) + btnH,
			}
			if f.hoveredRow == -1 && pt.X >= btnRect.Left && pt.X <= btnRect.Right && pt.Y >= btnRect.Top && pt.Y <= btnRect.Bottom {
				f.hoveredRow = 99
			}
			currY += btnH + scale(6)
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

	var bmi winapi.BITMAPINFO
	bmi.BmiHeader.BiSize = uint32(unsafe.Sizeof(bmi.BmiHeader))
	bmi.BmiHeader.BiWidth = w
	bmi.BmiHeader.BiHeight = -h // Negative height creates top-down DIB
	bmi.BmiHeader.BiPlanes = 1
	bmi.BmiHeader.BiBitCount = 32
	bmi.BmiHeader.BiCompression = winapi.BI_RGB

	var pBits *byte
	memBmp := winapi.CreateDIBSection(hdc, &bmi, winapi.DIB_RGB_COLORS, &pBits, 0, 0)
	defer winapi.DeleteObject(memBmp)

	oldBmp := winapi.SelectObject(memDC, memBmp)
	defer winapi.SelectObject(memDC, oldBmp)

	// Initialize memory buffer with ambient acrylic color in light mode so GDI font
	// rasterization antialiases against the actual perceived glass luminance instead of pitch black.
	var bgCol uint32 = 0x00000000
	if !f.isDark {
		bgCol = 0x00EFECE9
	}
	bgRect := winapi.RECT{Left: 0, Top: 0, Right: w, Bottom: h}
	hbrBg := winapi.CreateSolidBrush(bgCol)
	winapi.FillRect(memDC, &bgRect, hbrBg)
	winapi.DeleteObject(hbrBg)

	dc := memDC

	pal := getThemePalette(f.isDark, f.accentR, f.accentG, f.accentB)
	accentRGB := uint32(f.accentR) | (uint32(f.accentG) << 8) | (uint32(f.accentB) << 16)

	winapi.SetBkMode(dc, winapi.TRANSPARENT)

	scale := func(v int32) int32 { return winapi.ScaleDpi(v, f.dpi) }

	// Typography setup (ANTIALIASED_QUALITY = 4: Grayscale antialiasing for clean rendering on transparent glass)
	const fontQuality = 4

	fontTitle := winapi.CreateFont(-scale(13), 0, 0, 0, 600, 0, 0, 0, 1, 0, 0, fontQuality, 0, "Segoe UI Variable Text")
	defer winapi.DeleteObject(fontTitle)

	fontBold := winapi.CreateFont(-scale(13), 0, 0, 0, 700, 0, 0, 0, 1, 0, 0, fontQuality, 0, "Segoe UI Variable Text")
	defer winapi.DeleteObject(fontBold)

	fontRegular := winapi.CreateFont(-scale(13), 0, 0, 0, 600, 0, 0, 0, 1, 0, 0, fontQuality, 0, "Segoe UI Variable Text")
	defer winapi.DeleteObject(fontRegular)

	fontSmall := winapi.CreateFont(-scale(11), 0, 0, 0, 450, 0, 0, 0, 1, 0, 0, fontQuality, 0, "Segoe UI Variable Text")
	defer winapi.DeleteObject(fontSmall)

	fontTag := winapi.CreateFont(-scale(10), 0, 0, 0, 600, 0, 0, 0, 1, 0, 0, fontQuality, 0, "Segoe UI Variable Text")
	defer winapi.DeleteObject(fontTag)

	sidePad := scale(12)

	// 3. Header: "Nodal" on left, [Icon + Adapter Name / Marquee] on same row right-aligned
	oldFont := winapi.SelectObject(dc, fontTitle)
	winapi.SetTextColor(dc, pal.TextMain)

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

		pillFillCol := uint32(0x00F0EFEF)
		pillBorderCol := uint32(0x00E0DEDC)
		if f.isDark {
			pillFillCol = 0x00262626
			pillBorderCol = 0x003A3A3A
		}
		pillRgn := winapi.CreateRoundRectRgn(startX-scale(8), scale(9), startX+totalRightW+scale(8), scale(35), scale(13), scale(13))
		hbrPill := winapi.CreateSolidBrush(pillFillCol)
		winapi.FillRgn(dc, pillRgn, hbrPill)
		winapi.DeleteObject(hbrPill)
		hbrPillBorder := winapi.CreateSolidBrush(pillBorderCol)
		winapi.FrameRgn(dc, pillRgn, hbrPillBorder, 1, 1)
		winapi.DeleteObject(hbrPillBorder)
		winapi.DeleteObject(pillRgn)

		// Draw adapter icon glyph
		fontGlyph := winapi.CreateFont(-scale(13), 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, fontQuality, 0, "Segoe Fluent Icons")
		winapi.SelectObject(dc, fontGlyph)
		winapi.SetTextColor(dc, pal.TextSub)
		glyphRect := winapi.RECT{Left: startX, Top: scale(10), Right: startX + glyphSize, Bottom: scale(34)}
		winapi.DrawText(dc, adapterGlyph, &glyphRect, 0x0025) // DT_CENTER | DT_VCENTER | DT_SINGLELINE
		winapi.DeleteObject(fontGlyph)

		// Draw adapter name
		winapi.SelectObject(dc, fontSmall)
		winapi.SetTextColor(dc, pal.TextSub)
		nameRect := winapi.RECT{Left: startX + glyphSize + glyphGap, Top: scale(10), Right: startX + glyphSize + glyphGap + textW, Bottom: scale(34)}
		winapi.DrawText(dc, modeText, &nameRect, 0x0024) // DT_VCENTER | DT_SINGLELINE
	} else {
		// Overflows -> Smooth infinite marquee ticker
		startX := w - (sidePad + scale(4)) - availRightW

		pillFillCol := uint32(0x00F0EFEF)
		pillBorderCol := uint32(0x00E0DEDC)
		if f.isDark {
			pillFillCol = 0x00262626
			pillBorderCol = 0x003A3A3A
		}
		pillRgn := winapi.CreateRoundRectRgn(startX-scale(8), scale(9), w-(sidePad+scale(4))+scale(8), scale(35), scale(13), scale(13))
		hbrPill := winapi.CreateSolidBrush(pillFillCol)
		winapi.FillRgn(dc, pillRgn, hbrPill)
		winapi.DeleteObject(hbrPill)
		hbrPillBorder := winapi.CreateSolidBrush(pillBorderCol)
		winapi.FrameRgn(dc, pillRgn, hbrPillBorder, 1, 1)
		winapi.DeleteObject(hbrPillBorder)
		winapi.DeleteObject(pillRgn)

		// Draw adapter icon glyph
		fontGlyph := winapi.CreateFont(-scale(13), 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, fontQuality, 0, "Segoe Fluent Icons")
		winapi.SelectObject(dc, fontGlyph)
		winapi.SetTextColor(dc, pal.TextSub)
		glyphRect := winapi.RECT{Left: startX, Top: scale(10), Right: startX + glyphSize, Bottom: scale(34)}
		winapi.DrawText(dc, adapterGlyph, &glyphRect, 0x0025) // DT_CENTER | DT_VCENTER | DT_SINGLELINE
		winapi.DeleteObject(fontGlyph)

		// Viewport for marquee text
		nameRect := winapi.RECT{Left: startX + glyphSize + glyphGap, Top: scale(10), Right: w - (sidePad + scale(4)), Bottom: scale(34)}

		winapi.SelectObject(dc, fontSmall)
		winapi.SetTextColor(dc, pal.TextSub)

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
		winapi.SetTextColor(dc, pal.TextSub)
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
			hbr := winapi.CreateSolidBrush(pal.HoverPill)
			winapi.FillRgn(dc, rgn, hbr)
			winapi.DeleteObject(hbr)
		}
		hbrBorder := winapi.CreateSolidBrush(pal.Border)
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
				hbrActiveFill := winapi.CreateSolidBrush(pal.ActivePillFill)
				winapi.FillRgn(dc, rgn, hbrActiveFill)
				winapi.DeleteObject(hbrActiveFill)

				// Delicate, subtle accent border
				hbrActiveBorder := winapi.CreateSolidBrush(pal.ActivePillBorder)
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
				hbrHov := winapi.CreateSolidBrush(pal.HoverPill)
				winapi.FillRgn(dc, rgn, hbrHov)
				winapi.DeleteObject(hbrHov)

				hbrBorder := winapi.CreateSolidBrush(pal.Border)
				winapi.FrameRgn(dc, rgn, hbrBorder, 1, 1)
				winapi.DeleteObject(hbrBorder)
				winapi.DeleteObject(rgn)
			} else {
				cardFillCol := uint32(0x00FAF9F8)
				cardBorderCol := uint32(0x00E2E0DE)
				if f.isDark {
					cardFillCol = 0x00202020
					cardBorderCol = 0x00323232
				}
				rgn := winapi.CreateRoundRectRgn(rowRect.Left, rowRect.Top, rowRect.Right, rowRect.Bottom, scale(6), scale(6))
				hbrCardFill := winapi.CreateSolidBrush(cardFillCol)
				winapi.FillRgn(dc, rgn, hbrCardFill)
				winapi.DeleteObject(hbrCardFill)

				hbrBorder := winapi.CreateSolidBrush(cardBorderCol)
				winapi.FrameRgn(dc, rgn, hbrBorder, 1, 1)
				winapi.DeleteObject(hbrBorder)
				winapi.DeleteObject(rgn)
			}

			// Right-aligned Tag Badge (Clean, minimal, non-distracting chip)
			maxTextRight := rowRect.Right - scale(12)
			if p.Tag != "" {
				winapi.SelectObject(dc, fontTag)
				tagCalcRect := winapi.RECT{Left: 0, Top: 0, Right: w, Bottom: scale(20)}
				winapi.DrawText(dc, p.Tag, &tagCalcRect, 0x0420) // DT_CALCRECT | DT_SINGLELINE
				tagTextW := tagCalcRect.Right - tagCalcRect.Left

				badgeH := scale(20)
				badgeW := tagTextW + scale(14)
				badgeRight := rowRect.Right - scale(10)
				badgeLeft := badgeRight - badgeW
				badgeTop := rowRect.Top + (rowRect.Height()-badgeH)/2
				badgeBottom := badgeTop + badgeH
				badgeRect := winapi.RECT{Left: badgeLeft, Top: badgeTop, Right: badgeRight, Bottom: badgeBottom}

				badgeRgn := winapi.CreateRoundRectRgn(badgeLeft, badgeTop, badgeRight, badgeBottom, scale(5), scale(5))

				tagBgCol := pal.TagBg
				tagBorderCol := pal.TagBorder
				tagTextCol := pal.TagText

				if isActive {
					tagBgCol = pal.ActivePillFill
					tagBorderCol = pal.ActivePillBorder
					tagTextCol = accentRGB
				} else if isHovered {
					tagBgCol = pal.TagHoverBg
					tagBorderCol = pal.TagHoverBorder
					tagTextCol = pal.TagHoverText
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
				winapi.SetTextColor(dc, pal.TextMain)
			} else {
				winapi.SelectObject(dc, fontRegular)
				winapi.SetTextColor(dc, pal.TextMain)
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
				winapi.SetTextColor(dc, pal.TextSub)

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
			btnH := scale(32)
			btnRect := winapi.RECT{
				Left:   sidePad,
				Top:    currY + scale(4),
				Right:  w - sidePad,
				Bottom: currY + scale(4) + btnH,
			}
			isHovered := f.hoveredRow == 99
			rgn := winapi.CreateRoundRectRgn(btnRect.Left, btnRect.Top, btnRect.Right, btnRect.Bottom, scale(5), scale(5))

			btnBgCol := accentRGB
			if isHovered {
				hoverR := uint8(float64(f.accentR)*0.85 + 255*0.15)
				hoverG := uint8(float64(f.accentG)*0.85 + 255*0.15)
				hoverB := uint8(float64(f.accentB)*0.85 + 255*0.15)
				btnBgCol = uint32(hoverR) | (uint32(hoverG) << 8) | (uint32(hoverB) << 16)
			}

			hbrBtn := winapi.CreateSolidBrush(btnBgCol)
			winapi.FillRgn(dc, rgn, hbrBtn)
			winapi.DeleteObject(hbrBtn)

			borderR := uint8(float64(f.accentR) * 0.85)
			borderG := uint8(float64(f.accentG) * 0.85)
			borderB := uint8(float64(f.accentB) * 0.85)
			borderCol := uint32(borderR) | (uint32(borderG) << 8) | (uint32(borderB) << 16)
			hbrBorder := winapi.CreateSolidBrush(borderCol)
			winapi.FrameRgn(dc, rgn, hbrBorder, 1, 1)
			winapi.DeleteObject(hbrBorder)
			winapi.DeleteObject(rgn)

			fontButton := winapi.CreateFont(-scale(12), 0, 0, 0, 600, 0, 0, 0, 1, 0, 0, fontQuality, 0, "Segoe UI Variable Text")
			winapi.SelectObject(dc, fontButton)
			winapi.SetTextColor(dc, 0x00FFFFFF) // Pure white text
			winapi.DrawText(dc, "+ Custom DNS", &btnRect, 0x0001|0x0024) // DT_CENTER | DT_VCENTER | DT_SINGLELINE
			winapi.DeleteObject(fontButton)

			currY += btnH + scale(6)
		}
	}

	// 5. UAC Warning InfoBar (filled, WinUI 3-style)
	if f.uacAlert {
		bannerH := scale(36)
		bannerRect := winapi.RECT{
			Left:   sidePad,
			Top:    currY + scale(2),
			Right:  w - sidePad,
			Bottom: currY + scale(2) + bannerH,
		}

		// Filled rounded background
		rgn := winapi.CreateRoundRectRgn(bannerRect.Left, bannerRect.Top, bannerRect.Right, bannerRect.Bottom, scale(6), scale(6))
		hbrBg := winapi.CreateSolidBrush(pal.InfoBarBg)
		winapi.FillRgn(dc, rgn, hbrBg)
		winapi.DeleteObject(hbrBg)
		winapi.DeleteObject(rgn)

		// 3px left accent stripe
		stripeRgn := winapi.CreateRoundRectRgn(bannerRect.Left, bannerRect.Top, bannerRect.Left+scale(3), bannerRect.Bottom, scale(3), scale(3))
		hbrStripe := winapi.CreateSolidBrush(pal.InfoBarText)
		winapi.FillRgn(dc, stripeRgn, hbrStripe)
		winapi.DeleteObject(hbrStripe)
		winapi.DeleteObject(stripeRgn)

		// Warning glyph E7BA (Segoe Fluent Icons)
		const warningGlyph = "\uE7BA"
		fontGlyph := winapi.CreateFont(-scale(14), 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, fontQuality, 0, "Segoe Fluent Icons")
		winapi.SelectObject(dc, fontGlyph)
		winapi.SetTextColor(dc, pal.InfoBarText)
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
		winapi.SetTextColor(dc, pal.InfoBarText)
		txtRect := winapi.RECT{
			Left:   bannerRect.Left + scale(30),
			Top:    bannerRect.Top,
			Right:  bannerRect.Right - scale(4),
			Bottom: bannerRect.Bottom,
		}
		winapi.DrawText(dc, txt, &txtRect, 0x0024)
	}

	winapi.SelectObject(dc, oldFont)

	// In DWM Acrylic composition, standard GDI operations leave the alpha channel as 0x00.
	// For unpainted areas, alpha=0x00 is desirable because it allows the frosted Acrylic blur
	// and noise backdrop to show through. For any pixel painted with text, icons, or badges,
	// alpha must be 0xFF so DWM renders it opaquely rather than washing it out additively.
	// Post-processing DIB pixels:
	// Untouched background pixels matching bgCol are zeroed to 0x00000000 (Alpha=0) so DWM renders
	// the 100% clear frosted glass Acrylic blur & noise backdrop.
	// All painted content (antialiased text, cards, borders, icons) has its alpha forced to 0xFF (255)
	// so it renders with silky-smooth, razor-sharp edges over the live acrylic glass without any pixelation.
	if pBits != nil && w > 0 && h > 0 {
		pixelSlice := unsafe.Slice((*uint32)(unsafe.Pointer(pBits)), int(w*h))
		if !f.isDark {
			const (
				bgR         = 0xE9
				bgG         = 0xEC
				bgB         = 0xEF
				targetTextR = 0x1C
				targetTextG = 0x1C
				targetTextB = 0x1C
			)
			for i := 0; i < len(pixelSlice); i++ {
				pix := pixelSlice[i]
				rgb := pix & 0x00FFFFFF
				if rgb == bgCol {
					pixelSlice[i] = 0x00000000
					continue
				}

				b := int32(pix & 0xFF)
				g := int32((pix >> 8) & 0xFF)
				r := int32((pix >> 16) & 0xFF)

				diffR := bgR - r
				if diffR < 0 {
					diffR = -diffR
				}
				diffG := bgG - g
				if diffG < 0 {
					diffG = -diffG
				}
				diffB := bgB - b
				if diffB < 0 {
					diffB = -diffB
				}

				maxDiff := diffR
				if diffG > maxDiff {
					maxDiff = diffG
				}
				if diffB > maxDiff {
					maxDiff = diffB
				}

				if maxDiff <= 2 {
					pixelSlice[i] = 0x00000000
				} else if r > bgR || g > bgG || b > bgB || maxDiff > 160 {
					pixelSlice[i] = 0xFF000000 | rgb
				} else {
					alpha := (maxDiff * 255) / 180
					if alpha > 255 {
						alpha = 255
					}
					rPrem := (targetTextR * alpha) / 255
					gPrem := (targetTextG * alpha) / 255
					bPrem := (targetTextB * alpha) / 255
					pixelSlice[i] = (uint32(alpha) << 24) | (uint32(rPrem) << 16) | (uint32(gPrem) << 8) | uint32(bPrem)
				}
			}
		} else {
			for i := 0; i < len(pixelSlice); i++ {
				pix := pixelSlice[i]
				rgb := pix & 0x00FFFFFF
				if rgb == 0 {
					continue
				}
				b := pix & 0xFF
				g := (pix >> 8) & 0xFF
				r := (pix >> 16) & 0xFF
				maxC := r
				if g > maxC {
					maxC = g
				}
				if b > maxC {
					maxC = b
				}
				a := maxC
				if a > 255 || maxC >= 180 {
					a = 255
				}
				pixelSlice[i] = (a << 24) | rgb
			}
		}
	}

	winapi.BitBlt(hdc, 0, 0, w, h, memDC, 0, 0, winapi.SRCCOPY)
}
