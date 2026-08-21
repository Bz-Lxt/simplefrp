package idgen

import (
	"crypto/rand"
	"encoding/hex"
	"io"
)

func New(n int) string {
	if n <= 0 {
		n = 8
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return hex.EncodeToString([]byte("fallbackid"))[:min(n*2, 16)]
	}
	return hex.EncodeToString(b)
}

func SessionID() string { return New(4) }

func ClientNonce() string { return New(8) }

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
