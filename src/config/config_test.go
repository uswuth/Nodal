package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg == nil {
		t.Fatal("expected non-nil default config")
	}

	if cfg.Autostart != false {
		t.Errorf("expected autostart to default to false, got %v", cfg.Autostart)
	}

	if cfg.PrivacyMode != PrivacyModeVisible {
		t.Errorf("expected privacy_mode to default to visible, got %q", cfg.PrivacyMode)
	}

	if len(cfg.Profiles) != 3 {
		t.Fatalf("expected 3 default profiles, got %d", len(cfg.Profiles))
	}

	expectedNames := []string{"DHCP", "Cloudflare", "AdGuard DNS"}
	for i, name := range expectedNames {
		if cfg.Profiles[i].Name != name {
			t.Errorf("profile %d: expected name %q, got %q", i, name, cfg.Profiles[i].Name)
		}
	}

	if cfg.Profiles[0].Primary != "" {
		t.Errorf("DHCP primary should be empty, got %q", cfg.Profiles[0].Primary)
	}

	if cfg.Profiles[1].Primary != "1.1.1.1" || cfg.Profiles[1].Secondary != "1.0.0.1" {
		t.Errorf("Cloudflare servers mismatch: %s / %s", cfg.Profiles[1].Primary, cfg.Profiles[1].Secondary)
	}

	if cfg.Profiles[2].Primary != "94.140.14.14" || cfg.Profiles[2].Secondary != "94.140.15.15" {
		t.Errorf("AdGuard servers mismatch: %s / %s", cfg.Profiles[2].Primary, cfg.Profiles[2].Secondary)
	}
}

func TestAdvancedProfileFields(t *testing.T) {
	p := DNSProfile{
		Name:        "Custom Secure",
		Primary:     "1.1.1.1",
		Secondary:   "1.0.0.1",
		PrimaryV6:   "2606:4700:4700::1111",
		SecondaryV6: "2606:4700:4700::1001",
		DoT:         "1dot1dot1dot1.cloudflare-dns.com",
		DoH:         "https://cloudflare-dns.com/dns-query",
	}

	if p.PrimaryV6 != "2606:4700:4700::1111" || p.DoT != "1dot1dot1dot1.cloudflare-dns.com" {
		t.Errorf("unexpected custom profile advanced fields: %v", p)
	}
}

func TestPrivacyModes(t *testing.T) {
	cfg1 := &Config{PrivacyMode: "hidden"}
	if cfg1.EffectivePrivacyMode() != PrivacyModeHidden {
		t.Errorf("expected hidden, got %s", cfg1.EffectivePrivacyMode())
	}

	cfg2 := &Config{PrivacyMode: "masked"}
	if cfg2.EffectivePrivacyMode() != PrivacyModeMasked {
		t.Errorf("expected masked, got %s", cfg2.EffectivePrivacyMode())
	}

	cfg3 := &Config{PrivacyMode: "visible"}
	if cfg3.EffectivePrivacyMode() != PrivacyModeVisible {
		t.Errorf("expected visible, got %s", cfg3.EffectivePrivacyMode())
	}

	legacyTrue := true
	cfgLegacy := &Config{HideIP: &legacyTrue}
	if cfgLegacy.EffectivePrivacyMode() != PrivacyModeMasked {
		t.Errorf("expected masked for legacy hide_ip=true, got %s", cfgLegacy.EffectivePrivacyMode())
	}
}

func TestShowCustomButton(t *testing.T) {
	cfgDefault := &Config{}
	if !cfgDefault.EffectiveShowCustomButton() {
		t.Errorf("expected default EffectiveShowCustomButton to be true")
	}

	showFalse := false
	cfgFalse := &Config{ShowCustomButton: &showFalse}
	if cfgFalse.EffectiveShowCustomButton() {
		t.Errorf("expected EffectiveShowCustomButton to be false")
	}

	showTrue := true
	cfgTrue := &Config{ShowCustomButton: &showTrue}
	if !cfgTrue.EffectiveShowCustomButton() {
		t.Errorf("expected EffectiveShowCustomButton to be true")
	}
}

func TestConfigRoundTrip(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "nodal_cfg_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	filePath := filepath.Join(tempDir, "config.toml")
	cfg := DefaultConfig()
	cfg.Autostart = true
	cfg.PrivacyMode = PrivacyModeMasked

	err = saveConfigFileInternal(filePath, cfg)
	if err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Verify file was written
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("written config file is empty")
	}
}


