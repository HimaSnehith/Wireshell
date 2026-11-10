package tunnel

import (
	"strings"
	"testing"

	"github.com/wireshell/wireshell/pkg/config"
	"github.com/wireshell/wireshell/pkg/crypto"
)

func TestBuildUAPIConfig(t *testing.T) {
	priv, err := crypto.GeneratePrivateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	peerPriv, err := crypto.GeneratePrivateKey()
	if err != nil {
		t.Fatalf("GeneratePeerKey: %v", err)
	}
	peerPub := peerPriv.PublicKey()

	peers := []config.PeerConfig{
		{
			PublicKey:           peerPub.Base64(),
			AllowedIPs:          []string{"10.42.0.2/32"},
			Endpoint:            "203.0.113.1:51820",
			PersistentKeepalive: 25,
		},
	}

	uapi, err := BuildUAPIConfig(priv, 51820, peers)
	if err != nil {
		t.Fatalf("BuildUAPIConfig: %v", err)
	}

	if !strings.Contains(uapi, "private_key="+priv.Hex()) {
		t.Errorf("UAPI missing private key")
	}
	if !strings.Contains(uapi, "public_key="+peerPub.Hex()) {
		t.Errorf("UAPI missing peer public key")
	}
	if !strings.Contains(uapi, "allowed_ip=10.42.0.2/32") {
		t.Errorf("UAPI missing allowed_ip")
	}
	if !strings.Contains(uapi, "persistent_keepalive_interval=25") {
		t.Errorf("UAPI missing persistent keepalive")
	}
}

