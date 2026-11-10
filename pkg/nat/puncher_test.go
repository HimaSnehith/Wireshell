package nat

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestHolePuncherLocal(t *testing.T) {
	conn1, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("ListenUDP 1: %v", err)
	}
	defer conn1.Close()

	conn2, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("ListenUDP 2: %v", err)
	}
	defer conn2.Close()

	addr1 := conn1.LocalAddr().(*net.UDPAddr)
	addr2 := conn2.LocalAddr().(*net.UDPAddr)

	puncher1 := NewHolePuncher(addr1.Port)
	puncher2 := NewHolePuncher(addr2.Port)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	errChan := make(chan error, 2)
	winChan := make(chan *net.UDPAddr, 2)

	go func() {
		win, err := puncher1.Punch(ctx, conn1, []Endpoint{{IP: addr2.IP, Port: addr2.Port}})
		if err != nil {
			errChan <- err
		} else {
			winChan <- win
		}
	}()

	go func() {
		win, err := puncher2.Punch(ctx, conn2, []Endpoint{{IP: addr1.IP, Port: addr1.Port}})
		if err != nil {
			errChan <- err
		} else {
			winChan <- win
		}
	}()

	for i := 0; i < 2; i++ {
		select {
		case err := <-errChan:
			t.Fatalf("Punch failed: %v", err)
		case win := <-winChan:
			t.Logf("Punch succeeded with endpoint: %s", win.String())
		case <-time.After(3 * time.Second):
			t.Fatalf("Test timed out")
		}
	}
}
