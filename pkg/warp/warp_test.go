package warp

import (
	"context"
	"testing"
	"time"
)

func TestFetchCurrentIP(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	info, err := FetchCurrentIP(ctx)
	if err != nil {
		t.Fatalf("FetchCurrentIP failed: %v", err)
	}
	t.Logf("Current Public IP: %s (%s, %s)", info.Query, info.City, info.Country)
	if info.Query == "" {
		t.Errorf("Empty IP returned")
	}
}

func TestGenerateWarpConfig(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg, err := GenerateWarpConfig(ctx)
	if err != nil {
		t.Fatalf("GenerateWarpConfig failed: %v", err)
	}

	if cfg.PublicKey == "" || cfg.PrivateKey == "" {
		t.Errorf("Missing keys in WARP config")
	}
	if len(cfg.Peers) == 0 {
		t.Fatalf("No peers in WARP config")
	}
	if cfg.Peers[0].AllowedIPs[0] != "0.0.0.0/0" {
		t.Errorf("Expected 0.0.0.0/0 allowed IP, got %v", cfg.Peers[0].AllowedIPs)
	}
	t.Logf("WARP Config created successfully. Peer: %s, Endpoint: %s", cfg.Peers[0].PublicKey, cfg.Peers[0].Endpoint)
}

