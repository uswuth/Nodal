package menu

import (
	"errors"
	"fmt"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"nodal/src/winapi"

	"golang.org/x/sys/windows"
)

const (
	className = "NodalMenuFlyoutWindow"
)

type MenuItemType int

const (
	ItemAction MenuItemType = iota
	ItemSeparator
)

type MenuItem struct {
	Type     MenuItemType
	Text     string
	IconKind string
	OnClick  func()
}

type MenuFlyout struct {
	mu           sync.Mutex
	hwnd         windows.HWND
	visible      bool
	dpi          uint32
	isDark       bool
	hoveredIndex int
	openTime     time.Time
	items        []MenuItem
	onFlushDNS   func()
	onOpenConfig func()
	onExit       func()
}

var globalMenu *MenuFlyout

func NewMenuFlyout(onFlushDNS, onOpenConfig, onExit func()) (*MenuFlyout, error) {
	m := &MenuFlyout{
		dpi:          96,
		hoveredIndex: -1,
		onFlushDNS:   onFlushDNS,
		onOpenConfig: onOpenConfig,
		onExit:       onExit,
	}

	m.items = []MenuItem{
		{
			Type:     ItemAction,
			Text:     "Flush DNS cache",
			IconKind: "flush",
			OnClick: func() {
				if m.onFlushDNS != nil {
					m.onFlushDNS()
				}
			},
		},
		{
			Type:     ItemAction,
			Text:     "Open configuration",
			IconKind: "config",
			OnClick: func() {
				if m.onOpenConfig != nil {
					m.onOpenConfig()
				}
			},
		},
		{
			Type: ItemSeparator,
		},
		{
			Type:     ItemAction,
			Text:     "Exit Nodal",
			IconKind: "exit",
			OnClick: func() {
				if m.onExit != nil {
					m.onExit()
				}
			},
		},
	}

	if err := m.registerClass(); err != nil {
		return nil, err
	}

	if err := m.createWindow(); err != nil {
		return nil, err
	}

	m.updateTheme()
	return m, nil
}

func (m *MenuFlyout) registerClass() error {
	pClassName, _ := windows.UTF16PtrFromString(className)

	var wc winapi.WNDCLASSEXW
	wc.CbSize = uint32(unsafe.Sizeof(wc))
	wc.Style = 0
	wc.LpfnWndProc = syscall.NewCallback(menuWndProc)
	wc.HInstance = 0
	wc.HCursor = windows.Handle(winapi.LoadCursor(32512))
	wc.LpszClassName = pClassName

	_, err := winapi.RegisterClassEx(&wc)
	if err != nil && !errors.Is(err, windows.ERROR_CLASS_ALREADY_EXISTS) {
		return fmt.Errorf("failed to register menu window class: %w", err)
	}
	return nil
}

func menuWndProc(hwnd windows.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	if globalMenu != nil && globalMenu.hwnd == hwnd {
		return globalMenu.handleMessage(hwnd, msg, wParam, lParam)
	}
	return winapi.DefWindowProc(hwnd, msg, wParam, lParam)
}

func (m *MenuFlyout) createWindow() error {
	pClassName, _ := windows.UTF16PtrFromString(className)
	pWindowName, _ := windows.UTF16PtrFromString("Nodal Menu")

	const (
		WS_POPUP         = 0x80000000
		WS_EX_TOPMOST    = 0x00000008
		WS_EX_TOOLWINDOW = 0x00000080
	)

	hwnd, err := winapi.CreateWindowEx(
		WS_EX_TOPMOST|WS_EX_TOOLWINDOW,
		pClassName,
		pWindowName,
		WS_POPUP,
		0, 0, 100, 100,
		0, 0, 0, 0,
	)
	if err != nil {
		return fmt.Errorf("failed to create menu flyout window: %w", err)
	}

	m.hwnd = hwnd
	globalMenu = m
	m.dpi = winapi.GetDpiForHwnd(hwnd)
	return nil
}

func (m *MenuFlyout) updateTheme() {
	m.isDark = !winapi.GetAppsUseLightTheme()
	winapi.ApplyWindows11Styling(m.hwnd, m.isDark)
}

func (m *MenuFlyout) calculateDimensions() (w, h int32) {
	scale := func(v int32) int32 {
		return winapi.ScaleDpi(v, m.dpi)
	}

	w = scale(200)
	topPad := scale(6)
	bottomPad := scale(6)
	itemH := scale(34)
	sepH := scale(9)

	h = topPad + bottomPad
	for _, item := range m.items {
		if item.Type == ItemSeparator {
			h += sepH
		} else {
			h += itemH
		}
	}
	return w, h
}

func (m *MenuFlyout) Show(cursorPt winapi.POINT) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.updateTheme()
	m.repositionWindow(cursorPt)

	m.hoveredIndex = -1
	m.openTime = time.Now()
	m.visible = true

	winapi.ShowWindow(m.hwnd, winapi.SW_SHOW)
	winapi.SetForegroundWindow(m.hwnd)
	winapi.InvalidateRect(m.hwnd, nil, true)
}

func (m *MenuFlyout) Hide() {
	m.mu.Lock()
	m.hideUnlocked()
	m.mu.Unlock()
}

func (m *MenuFlyout) hideUnlocked() {
	if !m.visible {
		return
	}
	m.visible = false
	m.hoveredIndex = -1
	winapi.ShowWindow(m.hwnd, winapi.SW_HIDE)
}

func (m *MenuFlyout) repositionWindow(cursorPt winapi.POINT) {
	// Update DPI first
	m.dpi = winapi.GetDpiForHwnd(m.hwnd)
	w, h := m.calculateDimensions()

	anchorPt := cursorPt
	var iconRect winapi.RECT
	hasIconRect := false

	// Query the exact screen coordinates of the tray icon so flicking the mouse quickly won't throw off positioning.
	if rc, ok := winapi.GetTrayIconRect(0); ok {
		iconRect = rc
		hasIconRect = true
		anchorPt = winapi.POINT{
			X: rc.Left + rc.Width()/2,
			Y: rc.Top + rc.Height()/2,
		}
	}

	hMon := winapi.MonitorFromPoint(anchorPt, winapi.MONITOR_DEFAULTTONEAREST)
	var mi winapi.MONITORINFO
	winapi.GetMonitorInfo(hMon, &mi)
	workArea := mi.RcWork
	monArea := mi.RcMonitor

	margin := winapi.ScaleDpi(4, m.dpi)

	hTaskbar := winapi.FindWindow("Shell_TrayWnd", "")
	var rcTaskbar winapi.RECT
	hasTaskbar := hTaskbar != 0 && winapi.GetWindowRect(hTaskbar, &rcTaskbar)

	var posX, posY int32

	if hasIconRect {
		if hasTaskbar && rcTaskbar.Bottom > monArea.Top && rcTaskbar.Top < monArea.Bottom {
			if rcTaskbar.Top > monArea.Top+monArea.Height()/2 {
				// Taskbar at BOTTOM
				posX = anchorPt.X - w/2
				posY = iconRect.Top - h - margin
			} else if rcTaskbar.Bottom <= monArea.Top+monArea.Height()/2 {
				// Taskbar at TOP
				posX = anchorPt.X - w/2
				posY = iconRect.Bottom + margin
			} else if rcTaskbar.Left > monArea.Left+monArea.Width()/2 {
				// Taskbar at RIGHT
				posX = iconRect.Left - w - margin
				posY = anchorPt.Y - h/2
			} else {
				// Taskbar at LEFT
				posX = iconRect.Right + margin
				posY = anchorPt.Y - h/2
			}
		} else {
			posX = anchorPt.X - w/2
			posY = iconRect.Top - h - margin
		}
	} else if hasTaskbar && rcTaskbar.Bottom > monArea.Top && rcTaskbar.Top < monArea.Bottom {
		if rcTaskbar.Top > monArea.Top+monArea.Height()/2 {
			// Taskbar at BOTTOM
			posX = anchorPt.X - w/2
			posY = rcTaskbar.Top - h - margin
		} else if rcTaskbar.Bottom <= monArea.Top+monArea.Height()/2 {
			// Taskbar at TOP
			posX = anchorPt.X - w/2
			posY = rcTaskbar.Bottom + margin
		} else if rcTaskbar.Left > monArea.Left+monArea.Width()/2 {
			// Taskbar at RIGHT
			posX = rcTaskbar.Left - w - margin
			posY = anchorPt.Y - h/2
		} else {
			// Taskbar at LEFT
			posX = rcTaskbar.Right + margin
			posY = anchorPt.Y - h/2
		}
	} else if workArea.Bottom < monArea.Bottom {
		posX = anchorPt.X - w/2
		posY = workArea.Bottom - h - margin
	} else if workArea.Top > monArea.Top {
		posX = anchorPt.X - w/2
		posY = workArea.Top + margin
	} else {
		posX = anchorPt.X - w/2
		posY = monArea.Bottom - h - margin
	}

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

	winapi.SetWindowPos(m.hwnd, winapi.HWND_TOPMOST, posX, posY, w, h, winapi.SWP_SHOWWINDOW)
}

func (m *MenuFlyout) handleMessage(hwnd windows.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case winapi.WM_ACTIVATE:
		if (wParam & 0xFFFF) == winapi.WA_INACTIVE {
			// Post a message instead of calling Hide() directly from the wndproc
			// to avoid re-entrancy issues while the message pump is dispatching.
			winapi.PostMessage(hwnd, winapi.WM_USER+100, 0, 0)
		}
		return 0

	case winapi.WM_KILLFOCUS:
		winapi.PostMessage(hwnd, winapi.WM_USER+100, 0, 0)
		return 0

	case winapi.WM_USER + 100:
		m.Hide()
		return 0

	case winapi.WM_SETTINGCHANGE:
		m.updateTheme()
		winapi.InvalidateRect(m.hwnd, nil, true)
		return 0

	case winapi.WM_DPICHANGED:
		m.dpi = uint32(wParam & 0xFFFF)
		winapi.InvalidateRect(m.hwnd, nil, true)
		return 0

	case winapi.WM_ERASEBKGND:
		return 1

	case winapi.WM_PAINT:
		var ps winapi.PAINTSTRUCT
		hdc := winapi.BeginPaint(hwnd, &ps)
		m.paint(hdc)
		winapi.EndPaint(hwnd, &ps)
		return 0

	case winapi.WM_MOUSEMOVE:
		pt := winapi.POINT{
			X: int32(int16(lParam & 0xFFFF)),
			Y: int32(int16((lParam >> 16) & 0xFFFF)),
		}
		m.handleMouseMove(pt)
		return 0

	case winapi.WM_MOUSELEAVE:
		m.hoveredIndex = -1
		winapi.InvalidateRect(m.hwnd, nil, false)
		return 0

	case winapi.WM_LBUTTONUP:
		m.handleClick()
		return 0
	}

	return winapi.DefWindowProc(hwnd, msg, wParam, lParam)
}

func (m *MenuFlyout) handleMouseMove(pt winapi.POINT) {
	var tme winapi.TRACKMOUSEEVENT
	tme.CbSize = uint32(unsafe.Sizeof(tme))
	tme.DwFlags = winapi.TME_LEAVE
	tme.HwndTrack = m.hwnd
	winapi.TrackMouse(&tme)

	var rcClient winapi.RECT
	winapi.GetClientRect(m.hwnd, &rcClient)
	w := rcClient.Width()
	if w <= 0 {
		w, _ = m.calculateDimensions()
	}
	scale := func(v int32) int32 { return winapi.ScaleDpi(v, m.dpi) }

	sidePad := scale(5)
	topPad := scale(6)
	itemH := scale(34)
	sepH := scale(9)

	prev := m.hoveredIndex
	m.hoveredIndex = -1

	currY := topPad
	for i, item := range m.items {
		if item.Type == ItemSeparator {
			currY += sepH
			continue
		}

		rect := winapi.RECT{
			Left:   sidePad,
			Top:    currY,
			Right:  w - sidePad,
			Bottom: currY + itemH,
		}

		if pt.X >= rect.Left && pt.X <= rect.Right && pt.Y >= rect.Top && pt.Y <= rect.Bottom {
			m.hoveredIndex = i
		}
		currY += itemH
	}

	if prev != m.hoveredIndex {
		winapi.InvalidateRect(m.hwnd, nil, false)
	}
}

func (m *MenuFlyout) handleClick() {
	m.mu.Lock()
	idx := m.hoveredIndex
	m.hoveredIndex = -1
	var onClick func()
	if idx >= 0 && idx < len(m.items) {
		item := m.items[idx]
		if item.Type != ItemSeparator {
			m.hideUnlocked()
			onClick = item.OnClick
		}
	}
	m.mu.Unlock()

	if onClick != nil {
		go onClick()
	}
}

func (m *MenuFlyout) paint(hdc windows.Handle) {
	var rcClient winapi.RECT
	winapi.GetClientRect(m.hwnd, &rcClient)
	w := rcClient.Width()
	h := rcClient.Height()
	if w <= 0 || h <= 0 {
		w, h = m.calculateDimensions()
	}
	scale := func(v int32) int32 { return winapi.ScaleDpi(v, m.dpi) }

	memDC := winapi.CreateCompatibleDC(hdc)
	defer winapi.DeleteDC(memDC)

	memBmp := winapi.CreateCompatibleBitmap(hdc, w, h)
	defer winapi.DeleteObject(memBmp)

	oldBmp := winapi.SelectObject(memDC, memBmp)
	defer winapi.SelectObject(memDC, oldBmp)

	winapi.SetBkMode(memDC, winapi.TRANSPARENT)

	var (
		textMain  uint32 = 0xFFFFFF
		hoverPill uint32 = 0x333333
		sepCol    uint32 = 0x303030
	)

	if !m.isDark {
		textMain = 0x1A1A1A
		hoverPill = 0xEAEAEA
		sepCol = 0xEAEAEA
	}

	// 1. Frosted glass window background (0x000000 lets DWM Acrylic blur & noise show through)
	bgRect := winapi.RECT{Left: 0, Top: 0, Right: w, Bottom: h}
	hbrBg := winapi.CreateSolidBrush(0x000000)
	winapi.FillRect(memDC, &bgRect, hbrBg)
	winapi.DeleteObject(hbrBg)

	// 2. Fonts
	fontHandle := createMenuFont("Segoe UI Variable Text", 10, false, m.dpi)
	if fontHandle == 0 {
		fontHandle = createMenuFont("Segoe UI", 10, false, m.dpi)
	}
	defer winapi.DeleteObject(windows.Handle(fontHandle))

	fontIcons := createMenuFont("Segoe Fluent Icons", 12, false, m.dpi)
	if fontIcons == 0 {
		fontIcons = createMenuFont("Segoe MDL2 Assets", 12, false, m.dpi)
	}
	defer winapi.DeleteObject(windows.Handle(fontIcons))

	sidePad := scale(5)
	topPad := scale(6)
	itemH := scale(34)
	sepH := scale(9)

	currY := topPad
	for i, item := range m.items {
		if item.Type == ItemSeparator {
			// Separator line
			lineY := currY + sepH/2
			sepRect := winapi.RECT{
				Left:   sidePad + scale(10),
				Top:    lineY,
				Right:  w - sidePad - scale(10),
				Bottom: lineY + 1,
			}
			hbrSep := winapi.CreateSolidBrush(sepCol)
			winapi.FillRect(memDC, &sepRect, hbrSep)
			winapi.DeleteObject(hbrSep)
			currY += sepH
			continue
		}

		itemRect := winapi.RECT{
			Left:   sidePad,
			Top:    currY,
			Right:  w - sidePad,
			Bottom: currY + itemH,
		}

		isHovered := m.hoveredIndex == i
		if isHovered {
			pillRgn := winapi.CreateRoundRectRgn(itemRect.Left, itemRect.Top, itemRect.Right, itemRect.Bottom, scale(4), scale(4))
			hbrHov := winapi.CreateSolidBrush(hoverPill)
			winapi.FillRgn(memDC, pillRgn, hbrHov)
			winapi.DeleteObject(hbrHov)
			winapi.DeleteObject(pillRgn)
		}

		// Vector icon glyph
		var glyph string
		switch item.IconKind {
		case "flush":
			glyph = "\uE72C" // Refresh / Sync
		case "config":
			glyph = "\uE713" // Settings Gear
		case "exit":
			glyph = "\uE7E8" // Power
		}

		iconRect := winapi.RECT{
			Left:   itemRect.Left + scale(10),
			Top:    itemRect.Top,
			Right:  itemRect.Left + scale(28),
			Bottom: itemRect.Bottom,
		}
		winapi.SelectObject(memDC, fontIcons)
		winapi.SetTextColor(memDC, textMain)
		winapi.DrawText(memDC, glyph, &iconRect, 0x0001|0x0024) // DT_CENTER | DT_VCENTER | DT_SINGLELINE

		// Draw text
		textRect := winapi.RECT{
			Left:   itemRect.Left + scale(36),
			Top:    itemRect.Top,
			Right:  itemRect.Right - scale(10),
			Bottom: itemRect.Bottom,
		}
		winapi.SelectObject(memDC, fontHandle)
		winapi.SetTextColor(memDC, textMain)
		winapi.DrawText(memDC, item.Text, &textRect, 0x0024) // DT_VCENTER | DT_SINGLELINE

		currY += itemH
	}

	winapi.BitBlt(hdc, 0, 0, w, h, memDC, 0, 0, winapi.SRCCOPY)
}

func createMenuFont(face string, ptSize int32, bold bool, dpi uint32) windows.Handle {
	height := -int32(float64(ptSize) * float64(dpi) / 72.0)
	weight := int32(400)
	if bold {
		weight = 700
	}
	const CLEARTYPE_NATURAL_QUALITY = 6
	return winapi.CreateFont(
		height, 0, 0, 0, weight,
		0, 0, 0, 1, 0, 0,
		CLEARTYPE_NATURAL_QUALITY,
		0, face,
	)
}
