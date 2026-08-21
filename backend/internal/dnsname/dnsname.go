package dnsname

import (
	"fmt"
	"strings"
	"unicode"
)

func Normalize(name string) string {
	name = strings.TrimSpace(name)
	name = strings.TrimSuffix(name, ".")
	return strings.ToLower(name)
}

func Valid(name string) bool {
	name = Normalize(name)
	if name == "" || len(name) > 253 {
		return false
	}
	if name == "localhost" {
		return true
	}
	labels := strings.Split(name, ".")
	if len(labels) == 0 {
		return false
	}
	for _, label := range labels {
		if !validLabel(label) {
			return false
		}
	}
	return true
}

func validLabel(label string) bool {
	if label == "" || len(label) > 63 {
		return false
	}
	if label[0] == '-' || label[len(label)-1] == '-' {
		return false
	}
	for _, r := range label {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' {
			continue
		}
		return false
	}
	return true
}

func Split(name string) (labels []string, err error) {
	name = Normalize(name)
	if !Valid(name) {
		return nil, fmt.Errorf("invalid dns name %q", name)
	}
	return strings.Split(name, "."), nil
}

func Parent(name string) string {
	labels, err := Split(name)
	if err != nil || len(labels) < 2 {
		return ""
	}
	return strings.Join(labels[1:], ".")
}

func Join(labels []string) (string, error) {
	cleaned := make([]string, 0, len(labels))
	for _, label := range labels {
		label = strings.TrimSpace(label)
		if label == "" {
			continue
		}
		cleaned = append(cleaned, label)
	}
	name := strings.Join(cleaned, ".")
	if !Valid(name) {
		return "", fmt.Errorf("invalid dns name %q", name)
	}
	return Normalize(name), nil
}

func IsSubdomain(child, parent string) bool {
	child = Normalize(child)
	parent = Normalize(parent)
	if !Valid(child) || !Valid(parent) {
		return false
	}
	if child == parent {
		return true
	}
	return strings.HasSuffix(child, "."+parent)
}

func Match(name, pattern string) bool {
	name = Normalize(name)
	pattern = Normalize(pattern)
	if pattern == "*" {
		return Valid(name)
	}
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[2:]
		if name == suffix {
			return false
		}
		return IsSubdomain(name, suffix) && strings.Count(name, ".") == strings.Count(suffix, ".")+1
	}
	return name == pattern && Valid(name)
}

func HostFromAddr(addr string) string {
	addr = strings.TrimSpace(addr)
	if i := strings.LastIndex(addr, ":"); i > 0 && !strings.Contains(addr, "]") {
		if _, err := parsePort(addr[i+1:]); err == nil {
			addr = addr[:i]
		}
	}
	return Normalize(strings.Trim(addr, "[]"))
}

func parsePort(s string) (int, error) {
	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("not a port")
		}
		n = n*10 + int(r-'0')
		if n > 65535 {
			return 0, fmt.Errorf("port overflow")
		}
	}
	if n == 0 {
		return 0, fmt.Errorf("port 0")
	}
	return n, nil
}
