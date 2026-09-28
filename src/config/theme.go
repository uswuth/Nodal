package config

import (
	"nodal/src/winapi"
)

// ResolveIsDark evaluates a theme setting ("dark", "light", or "auto") against the system state.
func ResolveIsDark(themeSetting string) bool {
	switch themeSetting {
	case "dark":
		return true
	case "light":
		return false
	default:
		return !winapi.GetAppsUseLightTheme()
	}
}
