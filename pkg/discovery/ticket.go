package discovery

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/wireshell/wireshell/pkg/crypto"
	"github.com/wireshell/wireshell/pkg/nat"
)

const TicketPrefix = "wireshell://ticket/"

// PeerTicket contains everything needed to establish a P2P tunnel with a peer
type PeerTicket struct {
	Version   int            `json:"v"`
	PublicKey string         `json:"pub"`
	MeshIP    string         `json:"ip"`
	Port      int            `json:"port"`
	Endpoints []nat.Endpoint `json:"endpoints"`
	CreatedAt int64          `json:"ts"`
}

// CreateTicket builds a ticket from local node parameters
func CreateTicket(pubKey crypto.Key, meshIP string, port int, endpoints []nat.Endpoint) (*PeerTicket, error) {
	if _, _, err := net.ParseCIDR(meshIP); err != nil {
		if net.ParseIP(meshIP) == nil {
			return nil, fmt.Errorf("invalid mesh IP: %s", meshIP)
		}
	}

	return &PeerTicket{
		Version:   1,
		PublicKey: pubKey.Base64(),
		MeshIP:    meshIP,
		Port:      port,
		Endpoints: endpoints,
		CreatedAt: time.Now().Unix(),
	}, nil
}

// Encode converts the ticket to a compact shareable string (wireshell://ticket/...)
func (t *PeerTicket) Encode() (string, error) {
	data, err := json.Marshal(t)
	if err != nil {
		return "", fmt.Errorf("marshal ticket: %w", err)
	}
	b64 := base64.RawURLEncoding.EncodeToString(data)
	return TicketPrefix + b64, nil
}

// ParseTicket parses a ticket string (with or without wireshell://ticket/ prefix)
func ParseTicket(raw string) (*PeerTicket, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, TicketPrefix)

	data, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		// Fallback to standard base64
		data, err = base64.StdEncoding.DecodeString(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid base64 in ticket: %w", err)
		}
	}

	var ticket PeerTicket
	if err := json.Unmarshal(data, &ticket); err != nil {
		return nil, fmt.Errorf("invalid ticket json: %w", err)
	}

	if ticket.PublicKey == "" {
		return nil, fmt.Errorf("ticket missing public key")
	}
	if _, err := crypto.ParseKey(ticket.PublicKey); err != nil {
		return nil, fmt.Errorf("invalid public key in ticket: %w", err)
	}

	return &ticket, nil
}
