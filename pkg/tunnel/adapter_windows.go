//go:build windows

package tunnel

import (
	"fmt"
	"net/netip"

	"golang.zx2c4.com/wireguard/tun"
	"golang.zx2c4.com/wireguard/windows/tunnel/winipcfg"
)

// ConfigureInterface assigns the Mesh IP and subnet route to the Windows Wintun adapter
func ConfigureInterface(tunDev tun.Device, meshCIDR string) error {
	nativeTun, ok := tunDev.(*tun.NativeTun)
	if !ok {
		return fmt.Errorf("tun device is not *tun.NativeTun")
	}

	luid := winipcfg.LUID(nativeTun.LUID())

	prefix, err := netip.ParsePrefix(meshCIDR)
	if err != nil {
		return fmt.Errorf("invalid mesh CIDR %s: %w", meshCIDR, err)
	}

	// Set IP address on interface
	err = luid.SetIPAddresses([]netip.Prefix{prefix})
	if err != nil {
		return fmt.Errorf("failed to set IP address %s on Wintun: %w", meshCIDR, err)
	}

	// Add route for the mesh subnet
	subnetPrefix := netip.PrefixFrom(prefix.Masked().Addr(), prefix.Bits())
	_ = luid.AddRoute(subnetPrefix, netip.Addr{}, 0)

	return nil
}
