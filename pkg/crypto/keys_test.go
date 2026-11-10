package crypto

import (
	"testing"
)

func TestKeyGeneration(t *testing.T) {
	priv, err := GeneratePrivateKey()
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	// Verify clamping
	if priv[0]&7 != 0 {
		t.Errorf("Private key not properly clamped at byte 0")
	}
	if priv[31]&128 != 0 {
		t.Errorf("Private key bit 255 not cleared")
	}
	if priv[31]&64 == 0 {
		t.Errorf("Private key bit 254 not set")
	}

	pub := priv.PublicKey()
	if pub == [32]byte{} {
		t.Errorf("Derived public key should not be all zeros")
	}

	b64 := pub.Base64()
	if len(b64) != 44 {
		t.Errorf("Base64 string expected length 44, got %d", len(b64))
	}

	parsed, err := ParseKey(b64)
	if err != nil {
		t.Fatalf("Failed to parse base64 key: %v", err)
	}
	if parsed != pub {
		t.Errorf("Parsed key mismatch: expected %v, got %v", pub, parsed)
	}
}

