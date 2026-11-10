package nat

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"sync"
	"time"
)

const (
	ProbeMagic   = "WS_HOLE_PUNCH_V1:"
	ProbeReply   = "WS_HOLE_PUNCH_ACK:"
	ProbeTimeout = 5 * time.Second
)

// Candidate represents a candidate IP:port for connecting to a peer
type Candidate struct {
	Endpoint Endpoint `json:"endpoint"`
	IsLAN    bool     `json:"is_lan"`
}

// HolePuncher coordinates UDP hole punching between two peers behind NATs
type HolePuncher struct {
	localPort int
}

func NewHolePuncher(localPort int) *HolePuncher {
	return &HolePuncher{
		localPort: localPort,
	}
}

// Punch attempts to punch a bidirectional UDP hole to one of the target peer's candidates.
// Returns the winning resolved UDP endpoint.
func (h *HolePuncher) Punch(ctx context.Context, listenConn *net.UDPConn, targets []Endpoint) (*net.UDPAddr, error) {
	if len(targets) == 0 {
		return nil, fmt.Errorf("no target candidates provided")
	}

	cookieBytes := make([]byte, 8)
	if _, err := rand.Read(cookieBytes); err != nil {
		return nil, fmt.Errorf("generate cookie: %w", err)
	}
	cookie := hex.EncodeToString(cookieBytes)

	probeMsg := []byte(ProbeMagic + cookie)
	ackMsg := []byte(ProbeReply + cookie)

	ctxTimeout, cancel := context.WithTimeout(ctx, ProbeTimeout)
	defer cancel()

	resultChan := make(chan *net.UDPAddr, 1)
	var once sync.Once

	// Goroutine to listen for incoming hole-punch probes/acks on listenConn
	go func() {
		buf := make([]byte, 512)
		for {
			select {
			case <-ctxTimeout.Done():
				return
			default:
				_ = listenConn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
				n, raddr, err := listenConn.ReadFromUDP(buf)
				if err != nil {
					continue
				}

				data := buf[:n]
				// If peer sent a probe to us, respond with ACK and also declare connection
				if bytes.HasPrefix(data, []byte(ProbeMagic)) {
					peerCookie := bytes.TrimPrefix(data, []byte(ProbeMagic))
					reply := append([]byte(ProbeReply), peerCookie...)
					_, _ = listenConn.WriteToUDP(reply, raddr)

					once.Do(func() {
						resultChan <- raddr
					})
				} else if bytes.Equal(data, ackMsg) {
					// We received an ACK for our probe!
					once.Do(func() {
						resultChan <- raddr
					})
				}
			}
		}
	}()

	// Blast probes to all target endpoints periodically until established or timed out
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	// Resolve target UDP addrs
	targetAddrs := make([]*net.UDPAddr, 0, len(targets))
	for _, target := range targets {
		if addr, err := net.ResolveUDPAddr("udp4", target.String()); err == nil {
			targetAddrs = append(targetAddrs, addr)
		}
	}

	for {
		select {
		case <-ctxTimeout.Done():
			return nil, fmt.Errorf("hole punching timed out after %v", ProbeTimeout)
		case win := <-resultChan:
			return win, nil
		case <-ticker.C:
			for _, addr := range targetAddrs {
				_, _ = listenConn.WriteToUDP(probeMsg, addr)
			}
		}
	}
}

