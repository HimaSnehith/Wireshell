package tunnel

import (
	"fmt"
	"strings"

	"github.com/wireshell/wireshell/pkg/config"
	"github.com/wireshell/wireshell/pkg/crypto"
)

// BuildUAPIConfig creates the WireGuard UAPI configuration string for device initialization
func BuildUAPIConfig(privKey crypto.Key, listenPort int, peers []config.PeerConfig) (string, error) {
	var sb strings.Builder

	// Local interface configuration
	sb.WriteString(fmt.Sprintf("private_key=%s\n", privKey.Hex()))
	if listenPort > 0 {
		sb.WriteString(fmt.Sprintf("listen_port=%d\n", listenPort))
	}

	// Remote peers configuration
	for _, p := range peers {
		pubKey, err := crypto.ParseKey(p.PublicKey)
		if err != nil {
			return "", fmt.Errorf("invalid peer pubkey %s: %w", p.PublicKey, err)
		}

		sb.WriteString(fmt.Sprintf("public_key=%s\n", pubKey.Hex()))

		if p.PresharedKey != "" {
			psk, err := crypto.ParseKey(p.PresharedKey)
			if err != nil {
				return "", fmt.Errorf("invalid peer preshared key: %w", err)
			}
			sb.WriteString(fmt.Sprintf("preshared_key=%s\n", psk.Hex()))
		}

		if p.Endpoint != "" {
			sb.WriteString(fmt.Sprintf("endpoint=%s\n", p.Endpoint))
		}

		if p.PersistentKeepalive > 0 {
			sb.WriteString(fmt.Sprintf("persistent_keepalive_interval=%d\n", p.PersistentKeepalive))
		}

		for _, allowedIP := range p.AllowedIPs {
			sb.WriteString(fmt.Sprintf("allowed_ip=%s\n", allowedIP))
		}
	}

	return sb.String(), nil
}

// BuildUpdateEndpointUAPI creates a UAPI string to dynamically update a peer's endpoint
func BuildUpdateEndpointUAPI(peerPubKey crypto.Key, endpoint string) string {
	return fmt.Sprintf("public_key=%s\nupdate_only=true\nendpoint=%s\n", peerPubKey.Hex(), endpoint)
}
