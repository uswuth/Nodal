package elevation

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"

	"nodal/src/config"
	"nodal/src/dns"

	"golang.org/x/sys/windows"
)

const (
	TaskName = "Nodal"
)

var (
	ErrElevationRequired = errors.New("admin elevation is required to modify DNS settings")
	ErrUACCancelled      = errors.New("user declined UAC elevation prompt")
	ErrTimeoutWaiting    = errors.New("timed out waiting for elevated worker to apply DNS")
)

type PendingRequest struct {
	Profile config.DNSProfile `json:"profile"`
	Created time.Time         `json:"created"`
}

type WorkerResult struct {
	Success bool      `json:"success"`
	Error   string    `json:"error,omitempty"`
	Time    time.Time `json:"time"`
}

func GetLocalStateDir() (string, error) {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		var buf [windows.MAX_PATH]uint16
		_, err := windows.GetEnvironmentVariable(windows.StringToUTF16Ptr("LOCALAPPDATA"), &buf[0], uint32(len(buf)))
		if err != nil {
			return "", fmt.Errorf("failed to get LOCALAPPDATA: %w", err)
		}
		localAppData = windows.UTF16ToString(buf[:])
	}
	dir := filepath.Join(localAppData, "Nodal")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

// IsElevated checks whether the current process is running with administrative privileges.
func IsElevated() bool {
	var token windows.Token
	err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &token)
	if err != nil {
		return false
	}
	defer token.Close()

	var elevation uint32
	var retLen uint32
	err = windows.GetTokenInformation(
		token,
		windows.TokenElevation,
		(*byte)(unsafe.Pointer(&elevation)),
		uint32(unsafe.Sizeof(elevation)),
		&retLen,
	)
	if err != nil {
		return false
	}
	return elevation != 0
}

// IsTaskRegistered checks if the "Nodal" scheduled task exists in Task Scheduler.
func IsTaskRegistered() bool {
	cmd := exec.Command("schtasks.exe", "/query", "/tn", TaskName)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	err := cmd.Run()
	return err == nil
}

// TriggerUACRegistration requests one-time UAC elevation to register the scheduled task.
// If the user clicks "No" on UAC, returns ErrUACCancelled.
func TriggerUACRegistration() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	verb, _ := windows.UTF16PtrFromString("runas")
	file, _ := windows.UTF16PtrFromString(exePath)
	params, _ := windows.UTF16PtrFromString("--register-task")

	shell32 := windows.NewLazySystemDLL("shell32.dll")
	shellExecute := shell32.NewProc("ShellExecuteW")

	const SW_HIDE = 0
	r1, _, err := shellExecute.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(file)),
		uintptr(unsafe.Pointer(params)),
		0,
		uintptr(SW_HIDE),
	)

	// ShellExecute returns > 32 on success.
	// 1223 = ERROR_CANCELLED (user clicked "No" on UAC)
	if r1 <= 32 {
		if errors.Is(err, windows.ERROR_CANCELLED) || r1 == 1223 {
			return ErrUACCancelled
		}
		return fmt.Errorf("elevation failed (code %d): %w", r1, err)
	}

	// Give the elevated registration process up to 3 seconds to complete
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(200 * time.Millisecond)
		if IsTaskRegistered() {
			return nil
		}
	}

	if IsTaskRegistered() {
		return nil
	}
	return errors.New("scheduled task registration did not complete in time")
}

// RegisterTaskAsAdmin executes under --register-task when elevated.
func RegisterTaskAsAdmin() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	// Create scheduled task: runs <exePath> --worker with highest privileges on demand
	action := fmt.Sprintf("\"%s\" --worker", exePath)
	cmd := exec.Command(
		"schtasks.exe",
		"/create",
		"/tn", TaskName,
		"/tr", action,
		"/rl", "highest",
		"/f",
		"/sc", "ONCE",
		"/st", "00:00",
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("schtasks /create failed: %s: %w", string(out), err)
	}

	return nil
}

// ApplyProfileElevated coordinates the DNS change with the elevated background worker.
func ApplyProfileElevated(profile config.DNSProfile) error {
	// If the current process is already elevated (e.g. run directly as Admin), apply directly
	if IsElevated() {
		return dns.ApplyProfile(profile)
	}

	// Check if task exists; if not, request elevation registration
	if !IsTaskRegistered() {
		if err := TriggerUACRegistration(); err != nil {
			return err
		}
	}

	stateDir, err := GetLocalStateDir()
	if err != nil {
		return err
	}

	pendingFile := filepath.Join(stateDir, "pending.json")
	resultFile := filepath.Join(stateDir, "result.json")

	// Remove old result file if present
	_ = os.Remove(resultFile)

	// Write pending request
	req := PendingRequest{
		Profile: profile,
		Created: time.Now(),
	}
	reqData, err := json.Marshal(req)
	if err != nil {
		return err
	}
	if err := os.WriteFile(pendingFile, reqData, 0644); err != nil {
		return fmt.Errorf("failed to write pending DNS request: %w", err)
	}

	// Trigger the elevated task
	cmd := exec.Command("schtasks.exe", "/run", "/tn", TaskName)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Run(); err != nil {
		// Task might have been deleted externally per §9 edge cases
		_ = TriggerUACRegistration()
		return fmt.Errorf("failed to invoke elevated scheduled task (re-registering): %w", err)
	}

	// Wait for result (up to 2.5 seconds)
	deadline := time.Now().Add(2500 * time.Millisecond)
	for time.Now().Before(deadline) {
		time.Sleep(60 * time.Millisecond)
		if data, err := os.ReadFile(resultFile); err == nil && len(data) > 0 {
			var res WorkerResult
			if err := json.Unmarshal(data, &res); err == nil {
				if !res.Success {
					return fmt.Errorf("DNS change error: %s", res.Error)
				}
				return nil
			}
		}
	}

	return ErrTimeoutWaiting
}

// WorkerMain executes when launched as `--worker` by Task Scheduler with highest privileges.
func WorkerMain() int {
	stateDir, err := GetLocalStateDir()
	if err != nil {
		return 1
	}

	pendingFile := filepath.Join(stateDir, "pending.json")
	resultFile := filepath.Join(stateDir, "result.json")

	data, err := os.ReadFile(pendingFile)
	if err != nil {
		writeWorkerResult(resultFile, false, fmt.Sprintf("failed to read pending file: %v", err))
		return 1
	}

	var req PendingRequest
	if err := json.Unmarshal(data, &req); err != nil {
		writeWorkerResult(resultFile, false, fmt.Sprintf("invalid pending JSON: %v", err))
		return 1
	}

	// Apply DNS settings
	if err := dns.ApplyProfile(req.Profile); err != nil {
		writeWorkerResult(resultFile, false, err.Error())
		return 1
	}

	writeWorkerResult(resultFile, true, "")
	return 0
}

func writeWorkerResult(path string, success bool, errMsg string) {
	res := WorkerResult{
		Success: success,
		Error:   errMsg,
		Time:    time.Now(),
	}
	if data, err := json.Marshal(res); err == nil {
		_ = os.WriteFile(path, data, 0644)
	}
}
