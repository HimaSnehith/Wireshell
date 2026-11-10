//go:build windows

package mesh

import (
	"os/exec"
)

// EnableIPForwarding enables IP forwarding on Windows network interfaces
func EnableIPForwarding(tunInterface string) error {
	cmd := exec.Command("powershell", "-NoProfile", "-Command", "Set-NetIPInterface -Forwarding Enabled")
	return cmd.Run()
}

// DisableIPForwarding cleans up forwarding rules if needed
func DisableIPForwarding(tunInterface string) error {
	return nil
}

