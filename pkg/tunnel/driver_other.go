//go:build !windows

package tunnel

// EnsureWintun is a no-op on non-Windows platforms
func EnsureWintun() error {
	return nil
}
