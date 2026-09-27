package tray

import (
	"math"
	"unsafe"

	"nodal/src/winapi"

	"golang.org/x/sys/windows"
)

// CreateNodalIcon renders a crisp native Windows HICON faithful to dns-nodal-logo.svg.
// The SVG uses a 512×512 canvas; all coordinates are scaled proportionally to `size`.
// If withAlertBadge is true, an amber exclamation badge is drawn on the bottom-right.
func CreateNodalIcon(size int32, withAlertBadge bool) (windows.Handle, error) {
	if size <= 0 {
		size = int32(winapi.ScaleDpi(16, 96))
		if sm := winapi.GetSystemMetrics(winapi.SM_CXSMICON); sm > 0 {
			size = sm
		}
	}
	if size < 16 {
		size = 16
	}

	pixels := make([]byte, size*size*4)

	// setPixel writes a premultiplied-alpha BGRA pixel with alpha compositing.
	setPixel := func(x, y int32, r, g, b, a uint8) {
		if x < 0 || x >= size || y < 0 || y >= size {
			return
		}
		// DIB rows are bottom-up
		dy := size - 1 - y
		idx := (dy*size + x) * 4

		srcA := float64(a) / 255.0
		dstA := float64(pixels[idx+3]) / 255.0
		outA := srcA + dstA*(1.0-srcA)
		if outA > 0 {
			outR := (float64(r)*srcA + float64(pixels[idx+2])*dstA*(1.0-srcA)) / outA
			outG := (float64(g)*srcA + float64(pixels[idx+1])*dstA*(1.0-srcA)) / outA
			outB := (float64(b)*srcA + float64(pixels[idx+0])*dstA*(1.0-srcA)) / outA
			pixels[idx+0] = uint8(outB)
			pixels[idx+1] = uint8(outG)
			pixels[idx+2] = uint8(outR)
			pixels[idx+3] = uint8(outA * 255.0)
		}
	}

	// Scale and center: SVG content is centered at (256, 256) with a ~320 unit bounding box.
	// We map it to fill the icon canvas with a clean ~1px margin.
	scale := float64(size) / 324.0
	centerOffset := float64(size) / 2.0

	toPixelX := func(svgX float64) float64 {
		return (svgX-256.0)*scale + centerOffset
	}
	toPixelY := func(svgY float64) float64 {
		return (svgY-256.0)*scale + centerOffset
	}
	toPixelR := func(svgR float64) float64 {
		return svgR * scale
	}

	// SVG gradient: #3BA0F2 (59,160,242) at top-left, #0F6CBD (15,108,189) at bottom-right.
	lerpColor := func(t float64) (uint8, uint8, uint8) {
		t = math.Max(0, math.Min(1, t))
		return uint8(59*(1-t) + 15*t),
			uint8(160*(1-t) + 108*t),
			uint8(242*(1-t) + 189*t)
	}

	// drawCircle renders an anti-aliased filled circle using the logo gradient.
	drawCircle := func(cx, cy, r float64) {
		pcx := toPixelX(cx)
		pcy := toPixelY(cy)
		pr := toPixelR(r)
		minX := int32(math.Floor(pcx - pr - 1))
		maxX := int32(math.Ceil(pcx + pr + 1))
		minY := int32(math.Floor(pcy - pr - 1))
		maxY := int32(math.Ceil(pcy + pr + 1))
		for py := minY; py <= maxY; py++ {
			for px := minX; px <= maxX; px++ {
				dx := float64(px) + 0.5 - pcx
				dy := float64(py) + 0.5 - pcy
				dist := math.Sqrt(dx*dx + dy*dy)
				if dist <= pr+0.75 {
					t := ((dx + dy) / (2 * pr)) + 0.5
					cr, cg, cb := lerpColor(t)
					alpha := 255.0
					if dist > pr-0.75 {
						alpha = 255.0 * (pr + 0.75 - dist) / 1.5
					}
					setPixel(px, py, cr, cg, cb, uint8(alpha))
				}
			}
		}
	}

	// Two nodes — SVG: cx=170,342 cy=256 r=66
	drawCircle(170, 256, 66)
	drawCircle(342, 256, 66)

	// drawDot paints a filled circle for thick stroke simulation.
	drawDot := func(x, y, r float64, rv, gv, bv uint8, a uint8) {
		px := toPixelX(x)
		py := toPixelY(y)
		pr := toPixelR(r)
		minX := int32(math.Floor(px - pr - 1))
		maxX := int32(math.Ceil(px + pr + 1))
		minY := int32(math.Floor(py - pr - 1))
		maxY := int32(math.Ceil(py + pr + 1))
		for iy := minY; iy <= maxY; iy++ {
			for ix := minX; ix <= maxX; ix++ {
				dx := float64(ix) + 0.5 - px
				dy := float64(iy) + 0.5 - py
				dist := math.Sqrt(dx*dx + dy*dy)
				if dist <= pr+0.75 {
					aa := float64(a)
					if dist > pr-0.75 {
						aa = float64(a) * (pr + 0.75 - dist) / 1.5
					}
					setPixel(ix, iy, rv, gv, bv, uint8(aa))
				}
			}
		}
	}

	// drawBezierStroke rasterises a quadratic bezier as a series of dots.
	drawBezierStroke := func(p0x, p0y, p1x, p1y, p2x, p2y, strokeW float64, rv, gv, bv uint8) {
		halfW := strokeW / 2.0
		steps := int(math.Ceil(math.Sqrt(
			math.Pow(toPixelX(p2x)-toPixelX(p0x), 2)+math.Pow(toPixelY(p2y)-toPixelY(p0y), 2)) * 4))
		if steps < 25 {
			steps = 25
		}
		for i := 0; i <= steps; i++ {
			t := float64(i) / float64(steps)
			bx := (1-t)*(1-t)*p0x + 2*(1-t)*t*p1x + t*t*p2x
			by := (1-t)*(1-t)*p0y + 2*(1-t)*t*p1y + t*t*p2y
			drawDot(bx, by, halfW, rv, gv, bv, 230)
		}
	}

	// SVG arcs:
	//   Top:    M 208 200 Q 256 148 304 200   (A→B, arrowhead at ~304,200 pointing right)
	//   Bottom: M 304 312 Q 256 364 208 312   (B→A, arrowhead at ~208,312 pointing left)
	drawBezierStroke(208, 200, 256, 148, 304, 200, 22, 255, 255, 255)
	drawBezierStroke(304, 312, 256, 364, 208, 312, 22, 255, 255, 255)

	// Arrowheads — SVG defines filled white triangles:
	drawFilledTriangle := func(ax, ay, bx, by, cx, cy float64) {
		pax := toPixelX(ax)
		pay := toPixelY(ay)
		pbx := toPixelX(bx)
		pby := toPixelY(by)
		pcx := toPixelX(cx)
		pcy := toPixelY(cy)

		minX := int32(math.Floor(math.Min(pax, math.Min(pbx, pcx)) - 1))
		maxX := int32(math.Ceil(math.Max(pax, math.Max(pbx, pcx)) + 1))
		minY := int32(math.Floor(math.Min(pay, math.Min(pby, pcy)) - 1))
		maxY := int32(math.Ceil(math.Max(pay, math.Max(pby, pcy)) + 1))

		sign := func(p1x, p1y, p2x, p2y, p3x, p3y float64) float64 {
			return (p1x-p3x)*(p2y-p3y) - (p2x-p3x)*(p1y-p3y)
		}
		for iy := minY; iy <= maxY; iy++ {
			for ix := minX; ix <= maxX; ix++ {
				px := float64(ix) + 0.5
				py := float64(iy) + 0.5
				d1 := sign(px, py, pax, pay, pbx, pby)
				d2 := sign(px, py, pbx, pby, pcx, pcy)
				d3 := sign(px, py, pcx, pcy, pax, pay)
				hasNeg := (d1 < 0) || (d2 < 0) || (d3 < 0)
				hasPos := (d1 > 0) || (d2 > 0) || (d3 > 0)
				if !(hasNeg && hasPos) {
					setPixel(ix, iy, 255, 255, 255, 255)
				}
			}
		}
	}

	// Top arrowhead: M 294 188 L 312 200 L 294 212 Z
	drawFilledTriangle(292, 186, 314, 200, 292, 214)
	// Bottom arrowhead: M 218 324 L 200 312 L 218 300 Z
	drawFilledTriangle(220, 326, 198, 312, 220, 298)

	// Alert badge — amber circle with white exclamation mark on bottom-right corner.
	if withAlertBadge {
		badgeCX := toPixelX(512.0 - 100.0)
		badgeCY := toPixelY(512.0 - 100.0)
		badgeR := toPixelR(70.0)
		drawDot(badgeCX, badgeCY, badgeR, 247, 99, 12, 255)

		stemW := toPixelR(14.0)
		for dy := -toPixelR(36.0); dy <= toPixelR(8.0); dy += 1.5 {
			drawDot(badgeCX, badgeCY+dy, stemW/2, 255, 255, 255, 255)
		}
		drawDot(badgeCX, badgeCY+toPixelR(26.0), stemW/2, 255, 255, 255, 255)
	}

	// Assemble Win32 HICON from BGRA pixel buffer.
	hdcScreen := winapi.CreateCompatibleDC(0)
	defer winapi.DeleteDC(hdcScreen)

	var bmi winapi.BITMAPINFO
	bmi.BmiHeader.BiSize = uint32(unsafe.Sizeof(bmi.BmiHeader))
	bmi.BmiHeader.BiWidth = size
	bmi.BmiHeader.BiHeight = size
	bmi.BmiHeader.BiPlanes = 1
	bmi.BmiHeader.BiBitCount = 32
	bmi.BmiHeader.BiCompression = winapi.BI_RGB

	var pBits *byte
	hbmColor := winapi.CreateDIBSection(hdcScreen, &bmi, winapi.DIB_RGB_COLORS, &pBits, 0, 0)
	if hbmColor == 0 {
		return 0, windows.GetLastError()
	}
	defer winapi.DeleteObject(hbmColor)

	copy(unsafe.Slice(pBits, len(pixels)), pixels)

	maskBytes := make([]byte, ((size+15)/16*2)*size)
	hbmMask := winapi.CreateBitmap(size, size, 1, 1, unsafe.Pointer(&maskBytes[0]))
	defer winapi.DeleteObject(hbmMask)

	var ii winapi.ICONINFO
	ii.FIcon = 1
	ii.HbmMask = hbmMask
	ii.HbmColor = hbmColor

	return winapi.CreateIconIndirect(&ii)
}
