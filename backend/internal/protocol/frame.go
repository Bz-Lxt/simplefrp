package protocol

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"
)

const maxFrame = 64 * 1024

var (
	ErrFrameTooLarge = errors.New("protocol: frame too large")
	ErrEmptyFrame    = errors.New("protocol: empty frame")
)

func WriteFrame(w io.Writer, v any) error {
	body, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("protocol marshal: %w", err)
	}
	if len(body) == 0 {
		return ErrEmptyFrame
	}
	if len(body) > maxFrame {
		return ErrFrameTooLarge
	}
	hdr := make([]byte, 4)
	binary.BigEndian.PutUint32(hdr, uint32(len(body)))
	if _, err := w.Write(hdr); err != nil {
		return fmt.Errorf("protocol write header: %w", err)
	}
	if _, err := w.Write(body); err != nil {
		return fmt.Errorf("protocol write body: %w", err)
	}
	return nil
}

func ReadFrame(r io.Reader, v any) error {
	hdr := make([]byte, 4)
	if _, err := io.ReadFull(r, hdr); err != nil {
		return fmt.Errorf("protocol read header: %w", err)
	}
	n := binary.BigEndian.Uint32(hdr)
	if n == 0 {
		return ErrEmptyFrame
	}
	if n > maxFrame {
		return ErrFrameTooLarge
	}
	decoder := json.NewDecoder(io.LimitReader(r, int64(n)))
	if err := decoder.Decode(v); err != nil {
		return fmt.Errorf("protocol unmarshal: %w", err)
	}
	return nil
}

func ReadFrameDeadline(conn deadlineConn, v any, d time.Duration) error {
	if err := conn.SetDeadline(time.Now().Add(d)); err != nil {
		return err
	}
	defer conn.SetDeadline(time.Time{})
	return ReadFrame(conn, v)
}

type deadlineConn interface {
	io.Reader
	SetDeadline(time.Time) error
}
