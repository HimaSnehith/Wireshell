package mesh

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/wireshell/wireshell/pkg/config"
	"github.com/wireshell/wireshell/pkg/crypto"
	"github.com/wireshell/wireshell/pkg/nat"
	"github.com/wireshell/wireshell/pkg/tunnel"
)

// MeshEngine orchestrates tunnel lifecycle, hole punching, and self-healing peer connections
type MeshEngine struct {
	cfg        *config.Config
	tun        *tunnel.Tunnel
	puncher    *nat.HolePuncher
	candidates []nat.Endpoint
	mu         sync.Mutex
	cancel     context.CancelFunc
}

// NewMeshEngine creates a new MeshEngine
func NewMeshEngine(cfg *config.Config) *MeshEngine {
	return &MeshEngine{
		cfg:     cfg,
		puncher: nat.NewHolePuncher(cfg.ListenPort),
	}
}

// Start brings up the tunnel and starts background peer connection management
func (m *MeshEngine) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	ctx, cancel := context.WithCancel(ctx)
	m.cancel = cancel

	// 1. Discover local LAN IPs and STUN public WAN IP
	m.discoverEndpoints(ctx)

	// 2. Start the WireGuard tunnel
	tun, err := tunnel.NewTunnel(m.cfg)
	if err != nil {
		return fmt.Errorf("failed to start WireGuard tunnel: %w", err)
	}
	m.tun = tun

	// 3. For any peers configured with endpoints, start connecting
	for _, peer := range m.cfg.Peers {
		go m.connectPeer(ctx, peer)
	}

	// 4. Background health-check loop to self-heal broken connections
	go m.healthCheckLoop(ctx)

	return nil
}

// DiscoverEndpoints queries STUN and local interfaces
func (m *MeshEngine) discoverEndpoints(ctx context.Context) {
	var endpoints []nat.Endpoint

	// Local LAN IPs
	lanIPs, err := nat.GetLocalIPs()
	if err == nil {
		for _, ip := range lanIPs {
			endpoints = append(endpoints, nat.Endpoint{
				IP:   ip,
				Port: m.cfg.ListenPort,
			})
		}
	}

	// STUN public WAN IP
	stunCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if pubEp, err := nat.DiscoverPublicEndpoint(stunCtx, m.cfg.ListenPort); err == nil {
		endpoints = append(endpoints, *pubEp)
	}

	m.candidates = endpoints
}

// GetCandidates returns discovered candidates for this node
func (m *MeshEngine) GetCandidates() []nat.Endpoint {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.candidates
}

func (m *MeshEngine) connectPeer(ctx context.Context, peer config.PeerConfig) {
	if peer.Endpoint == "" {
		return
	}

	peerKey, err := crypto.ParseKey(peer.PublicKey)
	if err != nil {
		log.Printf("[mesh] Invalid public key for peer: %v", err)
		return
	}

	host, portStr, err := net.SplitHostPort(peer.Endpoint)
	if err != nil {
		log.Printf("[mesh] Invalid endpoint format %s: %v", peer.Endpoint, err)
		return
	}

	var port int
	_, _ = fmt.Sscanf(portStr, "%d", &port)
	ip := net.ParseIP(host)
	if ip == nil {
		return
	}

	target := nat.Endpoint{IP: ip, Port: port}

	// Listen connection for hole punching probe
	probeConn, err := net.ListenUDP("udp4", nil)
	if err != nil {
		return
	}
	defer probeConn.Close()

	punchCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()

	punchedAddr, err := m.puncher.Punch(punchCtx, probeConn, []nat.Endpoint{target})
	if err == nil && punchedAddr != nil {
		log.Printf("[mesh] Hole punch successful to %s -> %s", peerKey.ShortID(), punchedAddr.String())
		_ = m.tun.UpdatePeerEndpoint(peerKey, punchedAddr.String())
	}
}

func (m *MeshEngine) healthCheckLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stats, err := m.tun.GetPeerStats()
			if err != nil {
				continue
			}

			now := time.Now()
			for _, peer := range m.cfg.Peers {
				s, ok := stats[peer.PublicKey]
				if !ok || s.LastHandshake.IsZero() || now.Sub(s.LastHandshake) > 2*time.Minute {
					// Peer connection dropped or stale; attempt reconnect
					go m.connectPeer(ctx, peer)
				}
			}
		}
	}
}

// GetStats queries live statistics from the tunnel
func (m *MeshEngine) GetStats() (map[string]*tunnel.PeerStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.tun == nil {
		return nil, fmt.Errorf("tunnel not running")
	}
	return m.tun.GetPeerStats()
}

// Close stops the engine and shuts down the tunnel
func (m *MeshEngine) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cancel != nil {
		m.cancel()
	}
	if m.tun != nil {
		return m.tun.Close()
	}
	return nil
}
