package protocol_test

import (
	"bytes"
	"io"
	"net"
	"testing"
	"time"

	"simplefrp/internal/protocol"
)

func TestReadFrameDeadlineLeavesConnectionReusable(t *testing.T) {
	reader, writer := connectedTCPPair(t)
	defer reader.Close()
	defer writer.Close()

	want := protocol.AuthResponse{Type: protocol.TypeAuthOK, SessionID: "session-1"}
	if err := protocol.WriteFrame(writer, want); err != nil {
		t.Fatalf("write handshake frame: %v", err)
	}

	const handshakeTimeout = 25 * time.Millisecond
	var got protocol.AuthResponse
	if err := protocol.ReadFrameDeadline(reader, &got, handshakeTimeout); err != nil {
		t.Fatalf("read handshake frame: %v", err)
	}
	if got != want {
		t.Fatalf("handshake response = %+v, want %+v", got, want)
	}

	time.Sleep(2 * handshakeTimeout)
	payload := []byte("post-handshake traffic")
	if _, err := reader.Write(payload); err != nil {
		t.Fatalf("connection is not writable after successful handshake: %v", err)
	}
	if err := writer.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set peer read deadline: %v", err)
	}
	received := make([]byte, len(payload))
	if _, err := io.ReadFull(writer, received); err != nil {
		t.Fatalf("read post-handshake traffic: %v", err)
	}
	if !bytes.Equal(received, payload) {
		t.Fatalf("post-handshake traffic = %q, want %q", received, payload)
	}
}

func connectedTCPPair(t *testing.T) (net.Conn, net.Conn) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	accepted := make(chan net.Conn, 1)
	acceptErr := make(chan error, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			acceptErr <- err
			return
		}
		accepted <- conn
	}()

	dialed, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	select {
	case conn := <-accepted:
		return dialed, conn
	case err := <-acceptErr:
		dialed.Close()
		t.Fatalf("accept: %v", err)
	case <-time.After(time.Second):
		dialed.Close()
		t.Fatal("accept timed out")
	}
	return nil, nil
}
