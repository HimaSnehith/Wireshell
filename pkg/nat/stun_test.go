package nat

import (
	"context"
	"testing"
	"time"
)

func TestGetLocalIPs(t *testing.T) {
	ips, err := GetLocalIPs()
	if err != nil {
		t.Fatalf("Failed to get local IPs: %v", err)
	}
	t.Logf("Discovered %d local IP(s): %v", len(ips), ips)
}

func TestSTUNPublicDiscovery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ep, err := DiscoverPublicEndpoint(ctx, 0)
	if err != nil {
		t.Logf("STUN discovery note: %v", err)
		return
	}
	t.Logf("STUN discovered public endpoint: %s", ep.String())
	if ep.Port == 0 || ep.IP == nil {
		t.Errorf("Invalid STUN endpoint returned: %v", ep)
	}
}
