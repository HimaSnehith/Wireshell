package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/wireshell/wireshell/pkg/crypto"
)

const (
	DefaultWireGuardPort = 51820
	DefaultMTU           = 1420
	DefaultMeshCIDR      = "10.42.0.1/24"
	DefaultKeepaliveSec  = 25
)

// PeerConfig holds configuration for a remote WireGuard peer
type PeerConfig struct {
	PublicKey           string   `json:"public_key"`
	MeshIP              string   `json:"mesh_ip"`
	AllowedIPs          []string `json:"allowed_ips"`
	Endpoint            string   `json:"endpoint,omitempty"`
	PresharedKey        string   `json:"preshared_key,omitempty"`
	PersistentKeepalive int      `json:"persistent_keepalive"`
}

// Config represents the complete Wireshell node configuration
type Config struct {
	PrivateKey string       `json:"private_key"`
	PublicKey  string       `json:"public_key"`
	MeshIP     string       `json:"mesh_ip"`
	ListenPort int          `json:"listen_port"`
	MTU        int          `json:"mtu"`
	Peers      []PeerConfig `json:"peers"`
}

// DefaultConfigDir returns the OS-standard config directory for Wireshell
func DefaultConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("unable to find home directory: %w", err)
	}

	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData != "" {
			return filepath.Join(appData, "Wireshell"), nil
		}
	}
	return filepath.Join(home, ".wireshell"), nil
}

// DefaultConfigPath returns the full path to config.json
func DefaultConfigPath() (string, error) {
	dir, err := DefaultConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// GenerateNewConfig creates a fresh configuration with a new keypair
func GenerateNewConfig(meshIP string, port int) (*Config, error) {
	priv, err := crypto.GeneratePrivateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}
	pub := priv.PublicKey()

	if meshIP == "" {
		meshIP = DefaultMeshCIDR
	}
	if port <= 0 {
		port = DefaultWireGuardPort
	}

	return &Config{
		PrivateKey: priv.Base64(),
		PublicKey:  pub.Base64(),
		MeshIP:     meshIP,
		ListenPort: port,
		MTU:        DefaultMTU,
		Peers:      []PeerConfig{},
	}, nil
}

// Load reads a config from disk
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return &cfg, nil
}

// Save writes a config to disk with restricted permissions
func (c *Config) Save(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}

// AddOrUpdatePeer adds a peer or updates existing one
func (c *Config) AddOrUpdatePeer(peer PeerConfig) {
	if peer.PersistentKeepalive == 0 {
		peer.PersistentKeepalive = DefaultKeepaliveSec
	}

	for i, p := range c.Peers {
		if p.PublicKey == peer.PublicKey {
			c.Peers[i] = peer
			return
		}
	}
	c.Peers = append(c.Peers, peer)
}

// RemovePeer removes a peer by public key
func (c *Config) RemovePeer(publicKey string) bool {
	for i, p := range c.Peers {
		if p.PublicKey == publicKey {
			c.Peers = append(c.Peers[:i], c.Peers[i+1:]...)
			return true
		}
	}
	return false
}
