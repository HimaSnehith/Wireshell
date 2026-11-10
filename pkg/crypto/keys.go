package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/curve25519"
)

// Key represents a 32-byte WireGuard key (private, public, or pre-shared key)
type Key [32]byte

// GeneratePrivateKey generates a new Curve25519 private key clamped for WireGuard
func GeneratePrivateKey() (Key, error) {
	var key Key
	_, err := rand.Read(key[:])
	if err != nil {
		return Key{}, fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// WireGuard Curve25519 key clamping
	key[0] &= 248
	key[31] &= 127
	key[31] |= 64

	return key, nil
}

// PublicKey computes the Curve25519 public key corresponding to a private key
func (k Key) PublicKey() Key {
	var pub Key
	curve25519.ScalarBaseMult((*[32]byte)(&pub), (*[32]byte)(&k))
	return pub
}

// GeneratePresharedKey generates a 32-byte symmetric pre-shared key
func GeneratePresharedKey() (Key, error) {
	var key Key
	_, err := rand.Read(key[:])
	if err != nil {
		return Key{}, fmt.Errorf("failed to generate preshared key: %w", err)
	}
	return key, nil
}

// Base64 returns the base64-encoded string representation of the key
func (k Key) Base64() string {
	return base64.StdEncoding.EncodeToString(k[:])
}

// Hex returns the 64-character hex-encoded string used by WireGuard UAPI
func (k Key) Hex() string {
	return hex.EncodeToString(k[:])
}

// String implements fmt.Stringer
func (k Key) String() string {
	return k.Base64()
}

// ParseKey parses a base64-encoded 32-byte key
func ParseKey(s string) (Key, error) {
	var key Key
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return Key{}, fmt.Errorf("invalid base64 key: %w", err)
	}
	if len(b) != 32 {
		return Key{}, fmt.Errorf("key must be 32 bytes, got %d", len(b))
	}
	copy(key[:], b)
	return key, nil
}

// ShortID returns the first 8 characters of the base64 representation for clean logging
func (k Key) ShortID() string {
	s := k.Base64()
	if len(s) > 8 {
		return s[:8]
	}
	return s
}
