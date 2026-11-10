package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigLifeCycle(t *testing.T) {
	cfg, err := GenerateNewConfig("10.42.0.1/24", 51820)
	if err != nil {
		t.Fatalf("GenerateNewConfig: %v", err)
	}

	if cfg.PrivateKey == "" || cfg.PublicKey == "" {
		t.Fatalf("Empty keys in generated config")
	}

	peer := PeerConfig{
		PublicKey:  "mockPubKey123=",
		MeshIP:     "10.42.0.2/24",
		AllowedIPs: []string{"10.42.0.2/32"},
		Endpoint:   "203.0.113.1:51820",
	}

	cfg.AddOrUpdatePeer(peer)
	if len(cfg.Peers) != 1 {
		t.Fatalf("Expected 1 peer, got %d", len(cfg.Peers))
	}
	if cfg.Peers[0].PersistentKeepalive != DefaultKeepaliveSec {
		t.Fatalf("Expected default keepalive %d, got %d", DefaultKeepaliveSec, cfg.Peers[0].PersistentKeepalive)
	}

	tmpDir, err := os.MkdirTemp("", "wireshell-test-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.json")
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.PublicKey != cfg.PublicKey {
		t.Errorf("PublicKey mismatch: got %s, want %s", loaded.PublicKey, cfg.PublicKey)
	}

	removed := loaded.RemovePeer("mockPubKey123=")
	if !removed || len(loaded.Peers) != 0 {
		t.Fatalf("Failed to remove peer")
	}
}
