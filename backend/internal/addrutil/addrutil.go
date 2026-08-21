package addrutil

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

func SplitHostPort(addr string) (string, int, error) {
	host, port, err := net.SplitHostPort(strings.TrimSpace(addr))
	if err != nil {
		return "", 0, err
	}
	n, err := strconv.Atoi(port)
	if err != nil || n <= 0 || n > 65535 {
		return "", 0, fmt.Errorf("invalid port %q", port)
	}
	return host, n, nil
}

func JoinHostPort(host string, port int) string {
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		host = "[" + host + "]"
	}
	return net.JoinHostPort(host, strconv.Itoa(port))
}

func IsLoopback(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	ip := net.ParseIP(host)
	if ip != nil {
		return ip.IsLoopback()
	}
	return strings.EqualFold(host, "localhost")
}

func NormalizeBind(addr string, fallbackPort int) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return fmt.Sprintf("127.0.0.1:%d", fallbackPort)
	}
	if _, _, err := net.SplitHostPort(addr); err == nil {
		return addr
	}
	if port, err := strconv.Atoi(addr); err == nil && port > 0 && port <= 65535 {
		return fmt.Sprintf("127.0.0.1:%d", port)
	}
	return fmt.Sprintf("%s:%d", addr, fallbackPort)
}

func HostOnly(addr string) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(addr))
	if err != nil {
		return strings.TrimSpace(addr)
	}
	return host
}

func PortOnly(addr string) (int, error) {
	_, port, err := SplitHostPort(addr)
	return port, err
}

func SameEndpoint(a, b string) bool {
	ha, pa, errA := SplitHostPort(a)
	hb, pb, errB := SplitHostPort(b)
	if errA != nil || errB != nil {
		return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
	}
	return pa == pb && strings.EqualFold(ha, hb)
}

func MustPort(addr string, fallback int) int {
	if _, port, err := SplitHostPort(addr); err == nil {
		return port
	}
	if fallback <= 0 {
		return 0
	}
	return fallback
}
