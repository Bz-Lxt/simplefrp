package proxyhdr

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
)

const v1Prefix = "PROXY "

type Header struct {
	Proto      string
	SrcIP      net.IP
	DstIP      net.IP
	SrcPort    int
	DstPort    int
	Unknown    bool
	Raw        string
}

func ParseLine(line string) (*Header, error) {
	line = strings.TrimRight(line, "\r\n")
	if !strings.HasPrefix(line, v1Prefix) {
		return nil, fmt.Errorf("not a proxy v1 header")
	}
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return nil, fmt.Errorf("truncated proxy header")
	}
	if fields[1] == "UNKNOWN" {
		return &Header{Unknown: true, Raw: line}, nil
	}
	if len(fields) != 6 {
		return nil, fmt.Errorf("invalid proxy header field count")
	}
	proto := fields[1]
	if proto != "TCP4" && proto != "TCP6" {
		return nil, fmt.Errorf("unsupported proto %s", proto)
	}
	src := net.ParseIP(fields[2])
	dst := net.ParseIP(fields[3])
	if src == nil || dst == nil {
		return nil, fmt.Errorf("invalid proxy address")
	}
	sp, err := atoiPort(fields[4])
	if err != nil {
		return nil, err
	}
	dp, err := atoiPort(fields[5])
	if err != nil {
		return nil, err
	}
	return &Header{Proto: proto, SrcIP: src, DstIP: dst, SrcPort: sp, DstPort: dp, Raw: line}, nil
}

func Read(r *bufio.Reader) (*Header, error) {
	if err := peekPrefix(r); err != nil {
		return nil, err
	}
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	return ParseLine(line)
}

func peekPrefix(r *bufio.Reader) error {
	b, err := r.Peek(len(v1Prefix))
	if err != nil {
		return err
	}
	if string(b) != v1Prefix {
		return fmt.Errorf("missing PROXY prefix")
	}
	return nil
}

func (h *Header) SrcAddr() string {
	if h == nil || h.Unknown {
		return ""
	}
	return net.JoinHostPort(h.SrcIP.String(), strconv.Itoa(h.SrcPort))
}

func (h *Header) DstAddr() string {
	if h == nil || h.Unknown {
		return ""
	}
	return net.JoinHostPort(h.DstIP.String(), strconv.Itoa(h.DstPort))
}

func Format(src, dst net.Addr) string {
	sh, sp := split(src)
	dh, dp := split(dst)
	proto := "TCP4"
	if ip := net.ParseIP(sh); ip != nil && ip.To4() == nil {
		proto = "TCP6"
	}
	return fmt.Sprintf("PROXY %s %s %s %s %s\r\n", proto, sh, dh, sp, dp)
}

func Write(w io.Writer, src, dst net.Addr) error {
	_, err := io.WriteString(w, Format(src, dst))
	return err
}

func split(addr net.Addr) (string, string) {
	if addr == nil {
		return "0.0.0.0", "0"
	}
	host, port, err := net.SplitHostPort(addr.String())
	if err != nil {
		return "0.0.0.0", "0"
	}
	return host, port
}

func atoiPort(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 || n > 65535 {
		return 0, fmt.Errorf("invalid port %q", s)
	}
	return n, nil
}
