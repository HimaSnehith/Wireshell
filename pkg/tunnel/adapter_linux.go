//go:build linux

package tunnel

import (
	"fmt"
	"os/exec"

	"golang.zx2c4.com/wireguard/tun"
)

// ConfigureInterface assigns the Mesh IP and brings up the interface on Linux
func ConfigureInterface(tunDev tun.Device, meshCIDR string) error {
	name, err := tunDev.Name()
	if err != nil {
		return fmt.Errorf("get tun name: %w", err)
	}

	// ip addr add <meshCIDR> dev <name>
	cmdAddr := exec.Command("ip", "addr", "add", meshCIDR, "dev", name)
	if out, err := cmdAddr.CombinedOutput(); err != nil {
		return fmt.Errorf("ip addr add failed (%s): %w", string(out), err)
	}

	// ip link set up dev <name>
	cmdUp := exec.Command("ip", "link", "set", "up", "dev", name)
	if out, err := cmdUp.CombinedOutput(); err != nil {
		return fmt.Errorf("ip link set up failed (%s): %w", string(out), err)
	}

	return nil
}

