//go:build windows

package tunnel

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

//go:embed driver/wintun_amd64.dll
var wintunAMD64 []byte

//go:embed driver/wintun_arm64.dll
var wintunARM64 []byte

// EnsureWintun ensures that wintun.dll is available in the current working directory or executable directory
func EnsureWintun() error {
	// If wintun.dll already exists in current working dir, we are good
	if _, err := os.Stat("wintun.dll"); err == nil {
		return nil
	}

	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		targetInExeDir := filepath.Join(exeDir, "wintun.dll")
		if _, err := os.Stat(targetInExeDir); err == nil {
			return nil
		}
	}

	var dllData []byte
	switch runtime.GOARCH {
	case "amd64":
		dllData = wintunAMD64
	case "arm64":
		dllData = wintunARM64
	default:
		return fmt.Errorf("unsupported Windows architecture for embedded Wintun: %s", runtime.GOARCH)
	}

	// Write wintun.dll to the current working directory
	err = os.WriteFile("wintun.dll", dllData, 0755)
	if err != nil {
		// Fallback to temp directory if current directory is not writable
		tempDll := filepath.Join(os.TempDir(), "wintun.dll")
		if writeErr := os.WriteFile(tempDll, dllData, 0755); writeErr == nil {
			return nil
		}
		return fmt.Errorf("failed to write wintun.dll: %w", err)
	}

	return nil
}

