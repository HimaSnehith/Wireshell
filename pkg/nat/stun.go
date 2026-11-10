package nat

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/pion/stun/v3"
)

// DefaultSTUNServers lists fast, public, zero-authentication STUN servers
var DefaultSTUNServers = []string{
	"stun.l.google.com:19302",
	"stun1.l.google.com:19302",
	"stun.cloudflare.com:3478",
}

// Endpoint represents an IP and port pair
type Endpoint struct {
	IP   net.IP `json:"ip"`
	Port int    `json:"port"`
}

func (e Endpoint) String() string {
	return net.JoinHostPort(e.IP.String(), fmt.Sprintf("%d", e.Port))
}

// DiscoverPublicEndpoint queries public STUN servers to discover public IP & port
func DiscoverPublicEndpoint(ctx context.Context, localPort int) (*Endpoint, error) {
	var lastErr error

	for _, server := range DefaultSTUNServers {
		ep, err := querySTUN(ctx, server, localPort)
		if err == nil && ep != nil {
			return ep, nil
		}
		lastErr = err
	}

	return nil, fmt.Errorf("failed to discover public endpoint from STUN: %w", lastErr)
}

func querySTUN(ctx context.Context, server string, localPort int) (*Endpoint, error) {
	var localAddr *net.UDPAddr
	if localPort > 0 {
		localAddr = &net.UDPAddr{Port: localPort}
	}

	serverAddr, err := net.ResolveUDPAddr("udp4", server)
	if err != nil {
		return nil, fmt.Errorf("resolve server %s: %w", server, err)
	}

	conn, err := net.DialUDP("udp4", localAddr, serverAddr)
	if err != nil {
		return nil, fmt.Errorf("dial udp to %s: %w", server, err)
	}
	defer conn.Close()

	c, err := stun.NewClient(conn)
	if err != nil {
		return nil, fmt.Errorf("new stun client: %w", err)
	}
	defer c.Close()

	var xorAddr stun.XORMappedAddress
	var stunErr error

	message := stun.MustBuild(stun.TransactionID, stun.BindingRequest)

	ctxTimeout, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	done := make(chan struct{})

	err = c.Start(message, func(res stun.Event) {
		defer close(done)
		if res.Error != nil {
			stunErr = res.Error
			return
		}
		if getErr := xorAddr.GetFrom(res.Message); getErr != nil {
			stunErr = getErr
			return
		}
	})
	if err != nil {
		return nil, fmt.Errorf("start stun: %w", err)
	}

	select {
	case <-ctxTimeout.Done():
		return nil, fmt.Errorf("stun request timed out to %s", server)
	case <-done:
		if stunErr != nil {
			return nil, stunErr
		}
		return &Endpoint{
			IP:   xorAddr.IP,
			Port: xorAddr.Port,
		}, nil
	}
}

// GetLocalIPs returns non-loopback local IPv4 addresses of the host
func GetLocalIPs() ([]net.IP, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var ips []net.IP
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip != nil {
				ips = append(ips, ip)
			}
		}
	}
	return ips, nil
}
