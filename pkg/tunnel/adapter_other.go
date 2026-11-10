//go:build !windows && !linux

package tunnel

import (
	"fmt"

	"golang.zx2c4.com/wireguard/tun"
)

// ConfigureInterface fallback for other platforms
func ConfigureInterface(tunDev tun.Device, meshCIDR string) error {
	return fmt.Errorf("automatic interface IP configuration not implemented on this OS")
}

