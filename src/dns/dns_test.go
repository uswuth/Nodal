package dns

import (
	"nodal/src/config"
	"testing"
)

func TestDetectCurrentProfile(t *testing.T) {
	profiles := config.DefaultConfig().Profiles

	// If no match found or empty, defaults to DHCP
	profile := DetectCurrentProfile(profiles)
	if profile == "" {
		t.Error("expected non-empty profile name")
	}
}

func TestAdapterDisplayName(t *testing.T) {
	a1 := &AdapterInfo{
		IfType:       71,
		FriendlyName: "Wi-Fi",
		Description:  "Intel(R) Wi-Fi 6 AX201 160MHz",
		SSID:         "Home_5GHz",
	}
	if a1.DisplayName() != "Home_5GHz" {
		t.Errorf("expected SSID 'Home_5GHz', got %q", a1.DisplayName())
	}

	a2 := &AdapterInfo{
		IfType:       6,
		FriendlyName: "Ethernet",
		Description:  "Realtek Gaming 2.5GbE Family Controller",
	}
	if a2.DisplayName() != "Ethernet" {
		t.Errorf("expected friendly name 'Ethernet', got %q", a2.DisplayName())
	}
}


