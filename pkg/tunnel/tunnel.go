package tunnel

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun"

	"github.com/wireshell/wireshell/pkg/config"
	"github.com/wireshell/wireshell/pkg/crypto"
)

const InterfaceName = "wireshell0"

// PeerStatus tracks live metrics for a connected WireGuard peer
type PeerStatus struct {
	PublicKey           string    `json:"public_key"`
	Endpoint            string    `json:"endpoint"`
	LastHandshake       time.Time `json:"last_handshake"`
	BytesRx             uint64    `json:"bytes_rx"`
	BytesTx             uint64    `json:"bytes_tx"`
	PersistentKeepalive int       `json:"persistent_keepalive"`
	AllowedIPs          []string  `json:"allowed_ips"`
}

// Tunnel manages the WireGuard userspace device and TUN adapter
type Tunnel struct {
	dev       *device.Device
	tunDevice tun.Device
	cfg       *config.Config
}

// NewTunnel creates, configures, and starts a WireGuard tunnel
func NewTunnel(cfg *config.Config) (*Tunnel, error) {
	// 1. Ensure drivers (e.g. Wintun on Windows)
	if err := EnsureWintun(); err != nil {
		return nil, fmt.Errorf("failed to prepare tunnel driver: %w", err)
	}

	mtu := cfg.MTU
	if mtu <= 0 {
		mtu = config.DefaultMTU
	}

	// 2. Create the TUN interface
	tunDev, err := tun.CreateTUN(InterfaceName, mtu)
	if err != nil {
		return nil, fmt.Errorf("failed to create TUN device '%s': %w", InterfaceName, err)
	}

	// 3. Configure OS IP address and routes
	if err := ConfigureInterface(tunDev, cfg.MeshIP); err != nil {
		_ = tunDev.Close()
		return nil, fmt.Errorf("failed to configure network adapter IP: %w", err)
	}

	// 4. Initialize WireGuard userspace device
	logger := device.NewLogger(device.LogLevelError, fmt.Sprintf("[%s] ", InterfaceName))
	bind := conn.NewDefaultBind()
	dev := device.NewDevice(tunDev, bind, logger)

	// 5. Parse local private key
	privKey, err := crypto.ParseKey(cfg.PrivateKey)
	if err != nil {
		dev.Close()
		return nil, fmt.Errorf("invalid private key in config: %w", err)
	}

	// 6. Build UAPI config and push to device
	uapi, err := BuildUAPIConfig(privKey, cfg.ListenPort, cfg.Peers)
	if err != nil {
		dev.Close()
		return nil, fmt.Errorf("failed to build UAPI configuration: %w", err)
	}

	if err := dev.IpcSet(uapi); err != nil {
		dev.Close()
		return nil, fmt.Errorf("failed to configure WireGuard device: %w", err)
	}

	if err := dev.Up(); err != nil {
		dev.Close()
		return nil, fmt.Errorf("failed to bring up WireGuard device: %w", err)
	}

	return &Tunnel{
		dev:       dev,
		tunDevice: tunDev,
		cfg:       cfg,
	}, nil
}

// UpdatePeerEndpoint dynamically modifies a peer's endpoint (called after hole punching)
func (t *Tunnel) UpdatePeerEndpoint(peerPubKey crypto.Key, endpoint string) error {
	uapi := BuildUpdateEndpointUAPI(peerPubKey, endpoint)
	return t.dev.IpcSet(uapi)
}

// GetPeerStats queries the WireGuard device UAPI and returns status for all peers
func (t *Tunnel) GetPeerStats() (map[string]*PeerStatus, error) {
	statusStr, err := t.dev.IpcGet()
	if err != nil {
		return nil, fmt.Errorf("failed to read WireGuard status: %w", err)
	}

	stats := make(map[string]*PeerStatus)
	scanner := bufio.NewScanner(strings.NewReader(statusStr))

	var currentPeer *PeerStatus

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		val := parts[1]

		switch key {
		case "public_key":
			// Wireguard UAPI returns hex, convert to base64
			pubBytes, err := hexToBytes(val)
			var b64 string
			if err == nil {
				var k crypto.Key
				copy(k[:], pubBytes)
				b64 = k.Base64()
			} else {
				b64 = val
			}

			currentPeer = &PeerStatus{
				PublicKey: b64,
			}
			stats[b64] = currentPeer

		case "endpoint":
			if currentPeer != nil {
				currentPeer.Endpoint = val
			}
		case "last_handshake_time_sec":
			if currentPeer != nil {
				if sec, err := strconv.ParseInt(val, 10, 64); err == nil && sec > 0 {
					currentPeer.LastHandshake = time.Unix(sec, 0)
				}
			}
		case "rx_bytes":
			if currentPeer != nil {
				if rx, err := strconv.ParseUint(val, 10, 64); err == nil {
					currentPeer.BytesRx = rx
				}
			}
		case "tx_bytes":
			if currentPeer != nil {
				if tx, err := strconv.ParseUint(val, 10, 64); err == nil {
					currentPeer.BytesTx = tx
				}
			}
		case "persistent_keepalive_interval":
			if currentPeer != nil {
				if ka, err := strconv.Atoi(val); err == nil {
					currentPeer.PersistentKeepalive = ka
				}
			}
		case "allowed_ip":
			if currentPeer != nil {
				currentPeer.AllowedIPs = append(currentPeer.AllowedIPs, val)
			}
		}
	}

	return stats, nil
}

// Close gracefully closes the WireGuard device and network interface
func (t *Tunnel) Close() error {
	if t.dev != nil {
		t.dev.Close()
	}
	return nil
}

func hexToBytes(h string) ([]byte, error) {
	var b []byte
	_, err := fmt.Sscanf(h, "%x", &b)
	return b, err
}

func isElevated() bool {
	// Root check helper
	return os.Geteuid() == 0
}

