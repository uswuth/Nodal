package winapi

import (
	"testing"
)

func TestRectDimensions(t *testing.T) {
	r := RECT{Left: 10, Top: 20, Right: 110, Bottom: 170}
	if r.Width() != 100 {
		t.Errorf("expected width 100, got %d", r.Width())
	}
	if r.Height() != 150 {
		t.Errorf("expected height 150, got %d", r.Height())
	}
}

func TestScaleDpi(t *testing.T) {
	// At 100% DPI (96 dpi)
	if val := ScaleDpi(16, 96); val != 16 {
		t.Errorf("expected 16 at 96 DPI, got %d", val)
	}

	// At 150% DPI (144 dpi)
	if val := ScaleDpi(16, 144); val != 24 {
		t.Errorf("expected 24 at 144 DPI, got %d", val)
	}

	// At 200% DPI (192 dpi)
	if val := ScaleDpi(16, 192); val != 32 {
		t.Errorf("expected 32 at 192 DPI, got %d", val)
	}
}
