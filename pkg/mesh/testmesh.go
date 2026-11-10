package mesh

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/wireshell/wireshell/pkg/crypto"
	"github.com/wireshell/wireshell/pkg/discovery"
	"github.com/wireshell/wireshell/pkg/nat"
)

// TestMeshResult contains metrics from the simulated P2P mesh connection test
type TestMeshResult struct {
	NodeAPubKey string
	NodeBPubKey string
	STUNWAN     string
	TicketRaw   string
	PunchRTT    time.Duration
	PunchedAddr string
	Success     bool
}

// RunMeshTest runs a complete end-to-end simulated P2P connection test
func RunMeshTest(ctx context.Context) (*TestMeshResult, error) {
	result := &TestMeshResult{}

	// 1. Generate keys for two independent nodes
	privA, err := crypto.GeneratePrivateKey()
	if err != nil {
		return nil, fmt.Errorf("generate Key A: %w", err)
	}
	pubA := privA.PublicKey()
	result.NodeAPubKey = pubA.Base64()

	privB, err := crypto.GeneratePrivateKey()
	if err != nil {
		return nil, fmt.Errorf("generate Key B: %w", err)
	}
	pubB := privB.PublicKey()
	result.NodeBPubKey = pubB.Base64()

	// 2. STUN WAN Discovery
	stunCtx, stunCancel := context.WithTimeout(ctx, 4*time.Second)
	defer stunCancel()
	if pubEp, err := nat.DiscoverPublicEndpoint(stunCtx, 0); err == nil {
		result.STUNWAN = pubEp.String()
	} else {
		result.STUNWAN = "127.0.0.1:51820 (Local Offline Fallback)"
	}

	// 3. Open two local UDP sockets to simulate two devices
	connA, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		return nil, fmt.Errorf("listen UDP A: %w", err)
	}
	defer connA.Close()

	connB, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		return nil, fmt.Errorf("listen UDP B: %w", err)
	}
	defer connB.Close()

	addrA := connA.LocalAddr().(*net.UDPAddr)
	addrB := connB.LocalAddr().(*net.UDPAddr)

	// 4. Node A creates a real ticket containing its candidate addresses
	candidatesA := []nat.Endpoint{
		{IP: addrA.IP, Port: addrA.Port},
	}
	ticketA, err := discovery.CreateTicket(pubA, "10.42.0.1/24", addrA.Port, candidatesA)
	if err != nil {
		return nil, fmt.Errorf("create ticket A: %w", err)
	}

	encodedTicket, err := ticketA.Encode()
	if err != nil {
		return nil, fmt.Errorf("encode ticket A: %w", err)
	}
	result.TicketRaw = encodedTicket

	// 5. Node B parses Node A's ticket
	parsedTicketA, err := discovery.ParseTicket(encodedTicket)
	if err != nil {
		return nil, fmt.Errorf("node B parse ticket A: %w", err)
	}

	// 6. Both nodes initiate UDP hole punching concurrently
	puncherA := nat.NewHolePuncher(addrA.Port)
	puncherB := nat.NewHolePuncher(addrB.Port)

	start := time.Now()
	punchCtx, punchCancel := context.WithTimeout(ctx, 4*time.Second)
	defer punchCancel()

	winChan := make(chan *net.UDPAddr, 2)
	errChan := make(chan error, 2)

	go func() {
		target := []nat.Endpoint{{IP: addrB.IP, Port: addrB.Port}}
		win, err := puncherA.Punch(punchCtx, connA, target)
		if err != nil {
			errChan <- err
		} else {
			winChan <- win
		}
	}()

	go func() {
		target := parsedTicketA.Endpoints
		win, err := puncherB.Punch(punchCtx, connB, target)
		if err != nil {
			errChan <- err
		} else {
			winChan <- win
		}
	}()

	// Wait for both hole punches to complete
	for i := 0; i < 2; i++ {
		select {
		case err := <-errChan:
			return nil, fmt.Errorf("hole punch failed: %w", err)
		case win := <-winChan:
			result.PunchedAddr = win.String()
		case <-punchCtx.Done():
			return nil, fmt.Errorf("simulated mesh connection timed out")
		}
	}

	result.PunchRTT = time.Since(start)
	result.Success = true
	return result, nil
}

