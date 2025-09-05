package chal

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"io"
)

// Bytes returns a cryptographically secure random challenge of n bytes.
func Bytes(n int) []byte {
	b := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		panic(err)
	}
	return b
}

// B64 returns base64url (no padding) encoding of b.
func B64(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

// Hex returns lowercase hex encoding of b.
func Hex(b []byte) string { return hex.EncodeToString(b) }
