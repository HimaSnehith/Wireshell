package discovery

import (
	"net"
	"testing"

	"github.com/wireshell/wireshell/pkg/crypto"
	"github.com/wireshell/wireshell/pkg/nat"
)

func TestTicketEncodeDecode(t *testing.T) {
	priv, err := crypto.GeneratePrivateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	pub := priv.PublicKey()

	endpoints := []nat.Endpoint{
		{IP: net.ParseIP("192.168.1.100"), Port: 51820},
		{IP: net.ParseIP("203.0.113.50"), Port: 42100},
	}

	ticket, err := CreateTicket(pub, "10.42.0.2/24", 51820, endpoints)
	if err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}

	encoded, err := ticket.Encode()
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	t.Logf("Generated Ticket: %s", encoded)

	parsed, err := ParseTicket(encoded)
	if err != nil {
		t.Fatalf("ParseTicket: %v", err)
	}

	if parsed.PublicKey != pub.Base64() {
		t.Errorf("Pubkey mismatch: got %s, want %s", parsed.PublicKey, pub.Base64())
	}
	if parsed.MeshIP != "10.42.0.2/24" {
		t.Errorf("MeshIP mismatch: got %s, want 10.42.0.2/24", parsed.MeshIP)
	}
	if len(parsed.Endpoints) != 2 {
		t.Fatalf("Expected 2 endpoints, got %d", len(parsed.Endpoints))
	}
}
