//go:build !linux && !windows

package mesh

// EnableIPForwarding is a no-op on unsupported platforms
func EnableIPForwarding(tunInterface string) error {
	return nil
}

// DisableIPForwarding is a no-op on unsupported platforms
func DisableIPForwarding(tunInterface string) error {
	return nil
}

