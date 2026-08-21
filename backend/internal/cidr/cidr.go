package cidr

import (
	"fmt"
	"net"
	"strings"
)

type Net struct {
	IP   net.IP
	Mask net.IPMask
	raw  string
}

func Parse(s string) (*Net, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("empty cidr")
	}
	ip, ipnet, err := net.ParseCIDR(s)
	if err != nil {
		if parsed := net.ParseIP(s); parsed != nil {
			if v4 := parsed.To4(); v4 != nil {
				return &Net{IP: v4, Mask: net.CIDRMask(32, 32), raw: s}, nil
			}
			return &Net{IP: parsed, Mask: net.CIDRMask(128, 128), raw: s}, nil
		}
		return nil, err
	}
	return &Net{IP: ip, Mask: ipnet.Mask, raw: s}, nil
}

func MustParse(s string) *Net {
	n, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return n
}

func (n *Net) Contains(ip net.IP) bool {
	if n == nil || ip == nil {
		return false
	}
	ipnet := net.IPNet{IP: n.IP, Mask: n.Mask}
	return ipnet.Contains(ip)
}

func (n *Net) ContainsString(addr string) bool {
	host := addr
	if h, _, err := net.SplitHostPort(addr); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")
	return n.Contains(net.ParseIP(host))
}

func (n *Net) Overlaps(other *Net) bool {
	if n == nil || other == nil {
		return false
	}
	return n.Contains(other.IP) || other.Contains(n.IP)
}

func (n *Net) Network() net.IP {
	if n == nil {
		return nil
	}
	ip := n.IP
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}
	out := make(net.IP, len(ip))
	for i := range ip {
		if i < len(n.Mask) {
			out[i] = ip[i] & n.Mask[i]
		}
	}
	return out
}

func (n *Net) Broadcast() net.IP {
	if n == nil {
		return nil
	}
	ip := n.Network()
	if ip == nil {
		return nil
	}
	out := make(net.IP, len(ip))
	for i := range ip {
		mask := byte(0)
		if i < len(n.Mask) {
			mask = n.Mask[i]
		}
		out[i] = ip[i] | ^mask
	}
	return out
}

func (n *Net) Prefix() int {
	if n == nil {
		return 0
	}
	ones, _ := n.Mask.Size()
	return ones
}

func (n *Net) String() string {
	if n == nil {
		return ""
	}
	if n.raw != "" {
		ones, bits := n.Mask.Size()
		if ones == bits {
			return n.raw
		}
	}
	return n.Network().String() + "/" + itoa(n.Prefix())
}

func ParseList(items []string) ([]*Net, error) {
	out := make([]*Net, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		n, err := Parse(item)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, nil
}

func AnyContains(nets []*Net, ip net.IP) bool {
	for _, n := range nets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
