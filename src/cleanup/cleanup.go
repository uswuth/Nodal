// Package cleanup reverses every change Nodal makes to the machine so the application can be
// removed without leaving DNS overrides, scheduled tasks, registry entries or user data behind.
// It only ever touches objects that Nodal itself created.
package cleanup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"nodal/src/autostart"
	"nodal/src/dns"
	"nodal/src/elevation"
)

const (
	uninstallerName = "unins000.exe"
	localStateDir   = "Nodal"
	configDirName   = "nodal"
)

// Step is a single reversible action performed by a clean uninstall.
type Step struct {
	Name string
	Err  error
}

// Report is the outcome of a clean uninstall run.
type Report struct {
	Steps           []Step
	ExecutablePath  string
	UninstallerPath string // empty when Nodal runs as a portable copy that was never installed
}

// Run restores the machine to its pre-installation state in an order that keeps it usable at every
// point: DNS first, then the objects Nodal created, then the application itself.
func Run(removeUserData bool) *Report {
	report := &Report{}

	exePath, err := os.Executable()
	if err != nil {
		report.Steps = append(report.Steps, Step{"Locate the running executable", err})
	}
	report.ExecutablePath = exePath

	report.Steps = append(report.Steps,
		Step{"Restore automatic (DHCP) DNS on all adapters", dns.ResetAllToDHCP()},
		Step{"Remove the Windows startup entry", autostart.Remove(exePath)},
		Step{"Remove the privileged worker task", removeScheduledTask()},
		Step{"Remove temporary DNS hand-off files", removeLocalState()},
	)

	if removeUserData {
		report.Steps = append(report.Steps, Step{"Remove configuration and saved DNS profiles", removeConfig()})
	}

	report.UninstallerPath = findUninstaller(exePath)
	return report
}

// Failed returns the steps that did not complete.
func (r *Report) Failed() []Step {
	var failed []Step
	for _, step := range r.Steps {
		if step.Err != nil {
			failed = append(failed, step)
		}
	}
	return failed
}

// ExitCode is the process exit code of a clean uninstall: 1 when a step failed, otherwise 0.
func (r *Report) ExitCode() int {
	if len(r.Failed()) > 0 {
		return 1
	}
	return 0
}

// UserMessage returns the text to show once the cleanup finished. It is empty when the visible
// uninstaller already reports the outcome.
func (r *Report) UserMessage() string {
	failed := r.Failed()
	var b strings.Builder

	if len(failed) > 0 {
		b.WriteString("These steps could not be completed:\r\n\r\n")
		for _, step := range failed {
			fmt.Fprintf(&b, "  - %s\r\n    %v\r\n", step.Name, step.Err)
		}
		b.WriteString("\r\n")
	}

	if r.UninstallerPath != "" {
		if len(failed) > 0 {
			b.WriteString("The uninstaller will now run and remove the program files.")
		}
		return b.String()
	}

	if len(failed) == 0 {
		b.WriteString("Nodal runs as a portable copy, so it was never installed.\r\n\r\n")
		b.WriteString("All DNS overrides, startup entries, scheduled tasks, settings and temporary files were removed.\r\n\r\n")
	}
	b.WriteString("Delete this file to finish:\r\n")
	b.WriteString(r.ExecutablePath)
	return b.String()
}

// LaunchUninstaller starts the official uninstaller so the program files, shortcuts and the
// installed-apps registration are removed by the same code that created them. A two second delay
// gives this process time to exit and release the single-instance mutex the uninstaller checks.
func LaunchUninstaller(path string) error {
	const createNoWindow = 0x08000000

	// Keeping the call relative to the working directory avoids embedded quotes in the shell line.
	cmd := exec.Command("cmd.exe", "/c", "ping -n 3 127.0.0.1 >nul & "+filepath.Base(path)+" /NORESTART")
	cmd.Dir = filepath.Dir(path)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
	if err := cmd.Start(); err == nil {
		return nil
	}

	// Fall back to a direct start when the command interpreter is unavailable.
	direct := exec.Command(path, "/NORESTART")
	direct.Dir = filepath.Dir(path)
	direct.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return direct.Start()
}

func removeScheduledTask() error {
	if !elevation.IsTaskRegistered() {
		return nil
	}

	cmd := exec.Command("schtasks.exe", "/delete", "/tn", elevation.TaskName, "/f")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("schtasks /delete failed: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

func removeLocalState() error {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		return nil
	}
	return os.RemoveAll(filepath.Join(base, localStateDir))
}

// removeConfig deletes every directory GetConfigDir can resolve to, without recreating them.
func removeConfig() error {
	var failures []string

	var dirs []string
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		dirs = append(dirs, filepath.Join(home, ".config", configDirName))
	}
	if appData := os.Getenv("APPDATA"); appData != "" {
		dirs = append(dirs, filepath.Join(appData, configDirName))
	}

	for _, dir := range dirs {
		if err := os.RemoveAll(dir); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", dir, err))
		}
	}

	if len(failures) > 0 {
		return fmt.Errorf("%s", strings.Join(failures, "; "))
	}
	return nil
}

func findUninstaller(exePath string) string {
	if exePath == "" {
		return ""
	}
	path := filepath.Join(filepath.Dir(exePath), uninstallerName)
	if _, err := os.Stat(path); err != nil {
		return ""
	}
	return path
}
