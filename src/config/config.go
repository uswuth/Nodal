package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unsafe"

	"github.com/BurntSushi/toml"
	"golang.org/x/sys/windows"
)

const (
	PrivacyModeVisible = "visible"
	PrivacyModeMasked  = "masked"
	PrivacyModeHidden  = "hidden"
	MaxProfiles        = 5
)

type DNSProfile struct {
	Name        string `toml:"name" json:"name"`
	Tag         string `toml:"tag,omitempty" json:"tag,omitempty"`
	Primary     string `toml:"primary" json:"primary"`
	Secondary   string `toml:"secondary,omitempty" json:"secondary,omitempty"`
	PrimaryV6   string `toml:"primary_v6,omitempty" json:"primary_v6,omitempty"`
	SecondaryV6 string `toml:"secondary_v6,omitempty" json:"secondary_v6,omitempty"`
	DoT         string `toml:"dot,omitempty" json:"dot,omitempty"`
	DoH         string `toml:"doh,omitempty" json:"doh,omitempty"`
}

type Config struct {
	Autostart        bool         `toml:"autostart"`
	PrivacyMode      string       `toml:"privacy_mode"`
	Theme            string       `toml:"theme,omitempty"` // "auto" | "dark" | "light"
	ShowCustomButton *bool        `toml:"show_custom_button,omitempty"`
	HideIP           *bool        `toml:"hide_ip,omitempty"` // Legacy backward-compatibility
	Profiles         []DNSProfile `toml:"dns"`
}

func (c *Config) EffectiveTheme() string {
	t := strings.ToLower(strings.TrimSpace(c.Theme))
	if t == "dark" || t == "light" {
		return t
	}
	return "auto"
}

func (c *Config) EffectiveShowCustomButton() bool {
	if c.ShowCustomButton != nil {
		return *c.ShowCustomButton
	}
	return true
}

func (c *Config) EffectivePrivacyMode() string {
	mode := strings.ToLower(strings.TrimSpace(c.PrivacyMode))
	switch mode {
	case PrivacyModeMasked:
		return PrivacyModeMasked
	case PrivacyModeHidden:
		return PrivacyModeHidden
	case PrivacyModeVisible:
		return PrivacyModeVisible
	}

	// Legacy hide_ip fallback
	if c.HideIP != nil && *c.HideIP {
		return PrivacyModeMasked
	}
	return PrivacyModeVisible
}

var (
	mu             sync.RWMutex
	lastGoodConfig *Config
)

const schemaJSON = `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "Nodal Configuration",
  "description": "Configuration schema for Nodal DNS Switcher",
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "autostart": {
      "type": "boolean",
      "description": "Launch Nodal automatically when signing in to Windows",
      "default": false
    },
    "privacy_mode": {
      "type": "string",
      "enum": ["visible", "masked", "hidden"],
      "description": "DNS IP display mode:\n- 'visible': Show full IP addresses (1.1.1.1 • 1.0.0.1)\n- 'masked': Partially hide IP octets (1.1.*.* • 1.0.*.*)\n- 'hidden': Hide IP line completely, showing clean profile names only",
      "default": "visible"
    },
    "show_custom_button": {
      "type": "boolean",
      "description": "Show '+ Custom DNS' button in the flyout (set false for minimal UI)",
      "default": true
    },
    "hide_ip": {
      "type": "boolean",
      "description": "Legacy setting: true is equivalent to privacy_mode = 'masked'"
    },
    "dns": {
      "type": "array",
      "description": "DNS profiles list (Maximum 5 presets allowed)",
      "minItems": 1,
      "maxItems": 5,
      "items": {
        "type": "object",
        "additionalProperties": false,
        "required": ["name"],
        "properties": {
          "name": {
            "type": "string",
            "description": "Display name of the DNS profile (e.g. 'Cloudflare', 'Work DNS')"
          },
          "tag": {
            "type": "string",
            "description": "Short category or capability badge (e.g. 'Auto', 'Speed', 'AdBlock', 'Secure')"
          },
          "primary": {
            "type": "string",
            "description": "Primary IPv4 address (leave empty for DHCP)",
            "pattern": "^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$|^$"
          },
          "secondary": {
            "type": "string",
            "description": "Secondary IPv4 address (optional backup DNS server)",
            "pattern": "^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$|^$"
          },
          "primary_v6": {
            "type": "string",
            "description": "Primary IPv6 address (optional)"
          },
          "secondary_v6": {
            "type": "string",
            "description": "Secondary IPv6 address (optional)"
          },
          "dot": {
            "type": "string",
            "description": "DNS-over-TLS endpoint hostname (e.g. 'dns.adguard-dns.com', '1dot1dot1dot1.cloudflare-dns.com')"
          },
          "doh": {
            "type": "string",
            "description": "DNS-over-HTTPS endpoint URL (e.g. 'https://dns.adguard-dns.com/dns-query')"
          }
        }
      }
    }
  }
}`

func DefaultConfig() *Config {
	showCustom := true
	return &Config{
		Autostart:        false,
		PrivacyMode:      PrivacyModeVisible,
		Theme:            "auto",
		ShowCustomButton: &showCustom,
		Profiles: []DNSProfile{
			{
				Name:      "DHCP",
				Tag:       "Auto",
				Primary:   "",
				Secondary: "",
			},
			{
				Name:      "Cloudflare",
				Tag:       "Speed",
				Primary:   "1.1.1.1",
				Secondary: "1.0.0.1",
			},
			{
				Name:      "AdGuard DNS",
				Tag:       "AdBlock",
				Primary:   "94.140.14.14",
				Secondary: "94.140.15.15",
			},
		},
	}
}

func GetConfigDir() (string, error) {
	// Standard clean config directory: ~/.config/nodal/ (%USERPROFILE%\.config\nodal\)
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		dir := filepath.Join(home, ".config", "nodal")
		if err := os.MkdirAll(dir, 0755); err == nil {
			return dir, nil
		}
	}

	// Fallback to %APPDATA%\nodal
	appData := os.Getenv("APPDATA")
	if appData != "" {
		dir := filepath.Join(appData, "nodal")
		_ = os.MkdirAll(dir, 0755)
		return dir, nil
	}
	return "", fmt.Errorf("failed to determine config directory")
}

func GetConfigPath() (string, error) {
	dir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.toml"), nil
}

func EnsureSchemaFile() {
	dir, err := GetConfigDir()
	if err != nil {
		return
	}
	schemaPath := filepath.Join(dir, "schema.json")
	_ = os.WriteFile(schemaPath, []byte(schemaJSON), 0644)
}

// LoadConfig reads the TOML configuration from %APPDATA%\Nodal\config.toml.
// If the file does not exist or is empty (0 bytes), it writes the default profiles and returns them.
// If the file is malformed, it retains and returns the last known good configuration.
func LoadConfig() (*Config, error) {
	mu.Lock()
	defer mu.Unlock()

	EnsureSchemaFile()

	configPath, err := GetConfigPath()
	if err != nil {
		if lastGoodConfig != nil {
			return lastGoodConfig, nil
		}
		return DefaultConfig(), err
	}

	fi, err := os.Stat(configPath)
	if os.IsNotExist(err) || (err == nil && fi.Size() == 0) {
		cfg := DefaultConfig()
		if saveErr := saveConfigFileInternal(configPath, cfg); saveErr != nil {
			return cfg, saveErr
		}
		lastGoodConfig = cfg
		return cfg, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if lastGoodConfig != nil {
			return lastGoodConfig, nil
		}
		return DefaultConfig(), err
	}

	var parsed Config
	if _, err := toml.Decode(string(data), &parsed); err != nil {
		if lastGoodConfig != nil {
			return lastGoodConfig, fmt.Errorf("malformed TOML in %s, retaining current settings: %w", configPath, err)
		}
		cfg := DefaultConfig()
		lastGoodConfig = cfg
		return cfg, fmt.Errorf("malformed TOML in %s, using defaults: %w", configPath, err)
	}

	// Enforce maximum 5 presets
	if len(parsed.Profiles) > MaxProfiles {
		parsed.Profiles = parsed.Profiles[:MaxProfiles]
	}

	parsed.PrivacyMode = parsed.EffectivePrivacyMode()

	lastGoodConfig = &parsed
	return &parsed, nil
}

// MaskIP masks the last 2 octets of an IPv4 address for privacy (e.g. "192.168.1.1" -> "192.168.#.#")
func MaskIP(ip string) string {
	if ip == "" {
		return ""
	}
	parts := bytes.Split([]byte(ip), []byte("."))
	if len(parts) == 4 {
		return fmt.Sprintf("%s.%s.#.#", parts[0], parts[1])
	}
	return "########"
}

// SaveConfig saves the configuration to %APPDATA%\Nodal\config.toml.
func SaveConfig(cfg *Config) error {
	mu.Lock()
	defer mu.Unlock()

	EnsureSchemaFile()

	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	err = saveConfigFileInternal(configPath, cfg)
	if err == nil {
		lastGoodConfig = cfg
	}
	return err
}

func saveConfigFileInternal(path string, cfg *Config) error {
	var buf bytes.Buffer

	buf.WriteString("#:schema ./schema.json\n\n")
	buf.WriteString(fmt.Sprintf("autostart = %t\n", cfg.Autostart))
	buf.WriteString(fmt.Sprintf("theme = %q             # \"auto\" | \"dark\" | \"light\"\n", cfg.EffectiveTheme()))
	buf.WriteString(fmt.Sprintf("privacy_mode = %q      # \"visible\" | \"masked\" | \"hidden\"\n", cfg.EffectivePrivacyMode()))
	buf.WriteString(fmt.Sprintf("show_custom_button = %t    # show \"+ Custom DNS\" button in flyout\n", cfg.EffectiveShowCustomButton()))
	buf.WriteString("\n# DNS Presets (max 5 items)\n")

	limit := len(cfg.Profiles)
	if limit > MaxProfiles {
		limit = MaxProfiles
	}

	for _, p := range cfg.Profiles[:limit] {
		buf.WriteString("\n[[dns]]\n")
		buf.WriteString(fmt.Sprintf("name = %q\n", p.Name))
		if p.Tag != "" {
			buf.WriteString(fmt.Sprintf("tag = %q\n", p.Tag))
		}
		buf.WriteString(fmt.Sprintf("primary = %q\n", p.Primary))
		if p.Secondary != "" || p.Primary != "" {
			buf.WriteString(fmt.Sprintf("secondary = %q\n", p.Secondary))
		}
		if p.PrimaryV6 != "" {
			buf.WriteString(fmt.Sprintf("primary_v6 = %q\n", p.PrimaryV6))
		}
		if p.SecondaryV6 != "" {
			buf.WriteString(fmt.Sprintf("secondary_v6 = %q\n", p.SecondaryV6))
		}
		if p.DoT != "" {
			buf.WriteString(fmt.Sprintf("dot = %q\n", p.DoT))
		}
		if p.DoH != "" {
			buf.WriteString(fmt.Sprintf("doh = %q\n", p.DoH))
		}
	}

	return os.WriteFile(path, buf.Bytes(), 0644)
}

// OpenInEditor launches the user's default text editor for the config file.
func OpenInEditor() error {
	path, err := GetConfigPath()
	if err != nil {
		return err
	}

	// Ensure the config file and schema exist before trying to open it
	EnsureSchemaFile()
	fi, err := os.Stat(path)
	if os.IsNotExist(err) || (err == nil && fi.Size() == 0) {
		cfg := DefaultConfig()
		_ = saveConfigFileInternal(path, cfg)
	}

	verb, _ := windows.UTF16PtrFromString("open")
	file, _ := windows.UTF16PtrFromString(path)

	shell32 := windows.NewLazySystemDLL("shell32.dll")
	shellExecute := shell32.NewProc("ShellExecuteW")

	const SW_SHOWNORMAL = 1
	r1, _, err := shellExecute.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(file)),
		0,
		0,
		uintptr(SW_SHOWNORMAL),
	)
	if r1 <= 32 {
		return fmt.Errorf("failed to open config in editor: %v", err)
	}
	return nil
}
