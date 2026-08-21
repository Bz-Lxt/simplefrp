package protocol

import "errors"

const (
	TypeAuth     = "auth"
	TypeAuthOK   = "auth_ok"
	TypeAuthFail = "auth_fail"
)

var (
	ErrUnauthorized = errors.New("protocol: unauthorized")
	ErrBadAuthType  = errors.New("protocol: unexpected auth message type")
)

type AuthRequest struct {
	Type     string `json:"type"`
	Token    string `json:"token"`
	ClientID string `json:"client_id"`
}

type AuthResponse struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

func (r AuthRequest) Validate() error {
	if r.Type != TypeAuth {
		return ErrBadAuthType
	}
	if r.Token == "" {
		return ErrUnauthorized
	}
	if r.ClientID == "" {
		return errors.New("protocol: client_id required")
	}
	return nil
}
