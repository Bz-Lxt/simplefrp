package auth

import (
	"crypto/subtle"
	"errors"
	"strings"
)

var (
	ErrEmptyToken   = errors.New("auth: empty token")
	ErrTokenMismatch = errors.New("auth: token mismatch")
)

func Normalize(token string) string {
	return strings.TrimSpace(token)
}

func Equal(got, want string) error {
	got, want = Normalize(got), Normalize(want)
	if want == "" {
		return ErrEmptyToken
	}
	if subtle.ConstantTimeCompare([]byte(got), []byte(want)) != 1 {
		return ErrTokenMismatch
	}
	return nil
}

func LooksLikeToken(token string) bool {
	t := Normalize(token)
	if len(t) < 8 || len(t) > 128 {
		return false
	}
	for _, r := range t {
		if r < 33 || r > 126 {
			return false
		}
	}
	return true
}
