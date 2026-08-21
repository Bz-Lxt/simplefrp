package protocol

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

func TestWriteReadFrameRoundTrip(t *testing.T) {
	type payload struct {
		Type  string `json:"type"`
		Token string `json:"token"`
	}
	var buf bytes.Buffer
	in := payload{Type: "auth", Token: "secret"}
	if err := WriteFrame(&buf, in); err != nil {
		t.Fatal(err)
	}
	var out payload
	if err := ReadFrame(&buf, &out); err != nil {
		t.Fatal(err)
	}
	if out != in {
		t.Fatalf("got %+v want %+v", out, in)
	}
}

func TestReadFrameRejectsTooLarge(t *testing.T) {
	var buf bytes.Buffer
	buf.Write([]byte{0x00, 0x02, 0x00, 0x00}) // 131072 > maxFrame
	var out map[string]any
	if err := ReadFrame(&buf, &out); !errors.Is(err, ErrFrameTooLarge) {
		t.Fatalf("got %v want ErrFrameTooLarge", err)
	}
}

func TestReadFramePartialHeader(t *testing.T) {
	var out map[string]any
	err := ReadFrame(bytes.NewReader([]byte{0x00, 0x00}), &out)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		if err.Error() == "" {
			t.Fatal("empty error")
		}
	}
}

func TestAuthRequestValidate(t *testing.T) {
	tests := []struct {
		name    string
		req     AuthRequest
		wantErr error
	}{
		{name: "ok", req: AuthRequest{Type: TypeAuth, Token: "t", ClientID: "c"}},
		{name: "bad type", req: AuthRequest{Type: "nope", Token: "t", ClientID: "c"}, wantErr: ErrBadAuthType},
		{name: "empty token", req: AuthRequest{Type: TypeAuth, Token: "", ClientID: "c"}, wantErr: ErrUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr == nil && err != nil {
				t.Fatalf("unexpected %v", err)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("got %v want %v", err, tt.wantErr)
			}
		})
	}
}
