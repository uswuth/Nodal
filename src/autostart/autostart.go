package autostart

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

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

// Remove deletes the startup entry, but only while it still refers to the given executable.
// A value with the same name that belongs to another application is therefore never touched.
func Remove(exePath string) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	current, _, err := k.GetStringValue(valueName)
	if err != nil {
		return nil
	}

	target := strings.ToLower(filepath.Base(exePath))
	if target == "" || strings.ToLower(filepath.Base(strings.Trim(current, `"`))) != target {
		return nil
	}

	if err := k.DeleteValue(valueName); err != nil && !errors.Is(err, registry.ErrNotExist) {
		return err
	}
	return nil
}
