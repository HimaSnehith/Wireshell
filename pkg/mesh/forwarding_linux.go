//go:build linux

package mesh

import (
	"fmt"
	"os"
	"os/exec"
)

// EnableIPForwarding enables kernel IPv4 forwarding and iptables NAT masquerading on Linux
func EnableIPForwarding(tunInterface string) error {
	// 1. Enable sysctl net.ipv4.ip_forward = 1
	if err := os.WriteFile("/proc/sys/net/ipv4/ip_forward", []byte("1\n"), 0644); err != nil {
		// Fallback to sysctl command
		_ = exec.Command("sysctl", "-w", "net.ipv4.ip_forward=1").Run()
	}

	// 2. Add iptables FORWARD rule for wireshell interface
	cmdFwd := exec.Command("iptables", "-C", "FORWARD", "-i", tunInterface, "-j", "ACCEPT")
	if cmdFwd.Run() != nil {
		// Rule does not exist, append it
		_ = exec.Command("iptables", "-A", "FORWARD", "-i", tunInterface, "-j", "ACCEPT").Run()
	}

	// 3. Add iptables MASQUERADE for NAT out to the internet
	cmdNat := exec.Command("iptables", "-t", "nat", "-C", "POSTROUTING", "-o", "eth0", "-j", "MASQUERADE")
	if cmdNat.Run() != nil {
		_ = exec.Command("iptables", "-t", "nat", "-A", "POSTROUTING", "-j", "MASQUERADE").Run()
	}

	return nil
}

// DisableIPForwarding cleans up forwarding rules if needed
func DisableIPForwarding(tunInterface string) error {
	_ = exec.Command("iptables", "-D", "FORWARD", "-i", tunInterface, "-j", "ACCEPT").Run()
	_ = exec.Command("iptables", "-t", "nat", "-D", "POSTROUTING", "-j", "MASQUERADE").Run()
	return nil
}

