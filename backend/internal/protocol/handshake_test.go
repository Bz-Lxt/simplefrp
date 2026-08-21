package protocol

import (
	"net"
	"testing"
	"time"
)

func TestAuthHandshake(t *testing.T) {
	srv, cli := net.Pipe()
	defer srv.Close()
	defer cli.Close()

	errCh := make(chan error, 1)
	go func() {
		var req AuthRequest
		if err := ReadFrame(srv, &req); err != nil {
			errCh <- err
			return
		}
		if err := req.Validate(); err != nil {
			errCh <- err
			return
		}
		if req.Token != "tok" || req.ClientID != "c1" {
			errCh <- errString("token")
			return
		}
		errCh <- WriteFrame(srv, AuthResponse{Type: TypeAuthOK, SessionID: "s1"})
	}()

	if err := WriteFrame(cli, AuthRequest{Type: TypeAuth, Token: "tok", ClientID: "c1"}); err != nil {
		t.Fatal(err)
	}
	var resp AuthResponse
	if err := ReadFrameDeadline(cli, &resp, time.Second); err != nil {
		t.Fatal(err)
	}
	if resp.Type != TypeAuthOK || resp.SessionID != "s1" {
		t.Fatalf("resp=%+v", resp)
	}
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

type errString string

func (e errString) Error() string { return string(e) }
