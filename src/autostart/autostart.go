package autostart

import (
	"os"

	"golang.org/x/sys/windows/registry"
)

const (
	runKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`
	valueName  = "Nodal"
)

// Sync updates the Windows registry Run key based on the autostart flag.
func Sync(enabled bool) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	if enabled {
		exePath, err := os.Executable()
		if err != nil {
			return err
		}
		// Wrap in quotes in case of spaces in path
		cmd := `"` + exePath + `"`
		return k.SetStringValue(valueName, cmd)
	}

	// Delete if exists
	_, _, err = k.GetStringValue(valueName)
	if err == nil {
		_ = k.DeleteValue(valueName)
	}
	return nil
}

// IsEnabled checks if the autostart entry is currently present in the registry.
func IsEnabled() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()

	_, _, err = k.GetStringValue(valueName)
	return err == nil
}
