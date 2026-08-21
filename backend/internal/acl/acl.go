package acl

import (
	"net"
	"strings"
)

type List struct {
	nets []*net.IPNet
	ips  map[string]struct{}
	all  bool
}

func Parse(cidrs []string) (*List, error) {
	l := &List{ips: make(map[string]struct{})}
	if len(cidrs) == 0 {
		l.all = true
		return l, nil
	}
	for _, raw := range cidrs {
		raw = strings.TrimSpace(raw)
		if raw == "" || raw == "*" {
			l.all = true
			continue
		}
		if ip := net.ParseIP(raw); ip != nil {
			l.ips[ip.String()] = struct{}{}
			continue
		}
		_, n, err := net.ParseCIDR(raw)
		if err != nil {
			return nil, err
		}
		l.nets = append(l.nets, n)
	}
	return l, nil
}

func (l *List) Allow(addr net.Addr) bool {
	if l == nil || l.all {
		return true
	}
	host, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		host = addr.String()
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	if _, ok := l.ips[ip.String()]; ok {
		return true
	}
	for _, n := range l.nets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}
