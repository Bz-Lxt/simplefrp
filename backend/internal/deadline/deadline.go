package deadline

import (
	"net"
	"time"
)

func Set(conn net.Conn, d time.Duration) error {
	if conn == nil {
		return nil
	}
	if d <= 0 {
		return conn.SetDeadline(time.Time{})
	}
	return conn.SetDeadline(time.Now().Add(d))
}

func SetRead(conn net.Conn, d time.Duration) error {
	if conn == nil {
		return nil
	}
	if d <= 0 {
		return conn.SetReadDeadline(time.Time{})
	}
	return conn.SetReadDeadline(time.Now().Add(d))
}

func SetWrite(conn net.Conn, d time.Duration) error {
	if conn == nil {
		return nil
	}
	if d <= 0 {
		return conn.SetWriteDeadline(time.Time{})
	}
	return conn.SetWriteDeadline(time.Now().Add(d))
}

func Clear(conn net.Conn) error {
	if conn == nil {
		return nil
	}
	return conn.SetDeadline(time.Time{})
}
