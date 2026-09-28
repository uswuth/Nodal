package winapi

// CalculateFlyoutPosition calculates optimal screen coordinates (X, Y) for a flyout
// or popup window based on the notification tray icon position, monitor work area,
// and Windows taskbar orientation.
func CalculateFlyoutPosition(w, h int32, dpi uint32, fallbackPt POINT) (posX, posY int32) {
	anchorPt := fallbackPt
	var iconRect RECT
	hasIconRect := false

	if rc, ok := GetTrayIconRect(0); ok {
		iconRect = rc
		hasIconRect = true
		anchorPt = POINT{
			X: rc.Left + rc.Width()/2,
			Y: rc.Top + rc.Height()/2,
		}
	}

	hMon := MonitorFromPoint(anchorPt, MONITOR_DEFAULTTONEAREST)
	var mi MONITORINFO
	GetMonitorInfo(hMon, &mi)
	workArea := mi.RcWork
	monArea := mi.RcMonitor

	margin := ScaleDpi(4, dpi)

	hTaskbar := FindWindow("Shell_TrayWnd", "")
	var rcTaskbar RECT
	hasTaskbar := hTaskbar != 0 && GetWindowRect(hTaskbar, &rcTaskbar)

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

	// Clamp within monitor work area
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

	return posX, posY
}
