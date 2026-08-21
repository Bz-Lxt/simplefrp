package portset

import (
	"fmt"
	"sync"
)

type Set struct {
	mu       sync.Mutex
	min      int
	max      int
	next     int
	used     map[int]string
	reserved map[int]struct{}
}

func New(minPort, maxPort int) *Set {
	if minPort <= 0 {
		minPort = 20000
	}
	if maxPort <= minPort {
		maxPort = minPort + 1000
	}
	if maxPort > 65535 {
		maxPort = 65535
	}
	return &Set{
		min:      minPort,
		max:      maxPort,
		next:     minPort,
		used:     make(map[int]string),
		reserved: make(map[int]struct{}),
	}
}

func (s *Set) Reserve(port int) error {
	if port < s.min || port > s.max {
		return fmt.Errorf("port %d out of range %d-%d", port, s.min, s.max)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.used[port]; ok {
		return fmt.Errorf("port %d already allocated", port)
	}
	s.reserved[port] = struct{}{}
	return nil
}

func (s *Set) Alloc(owner string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	span := s.max - s.min + 1
	for i := 0; i < span; i++ {
		p := s.next
		s.next++
		if s.next > s.max {
			s.next = s.min
		}
		if _, ok := s.reserved[p]; ok {
			continue
		}
		if _, ok := s.used[p]; ok {
			continue
		}
		s.used[p] = owner
		return p, nil
	}
	return 0, fmt.Errorf("port pool exhausted")
}

func (s *Set) AllocExact(port int, owner string) error {
	if port < s.min || port > s.max {
		return fmt.Errorf("port %d out of range", port)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.reserved[port]; ok {
		return fmt.Errorf("port %d reserved", port)
	}
	if cur, ok := s.used[port]; ok {
		return fmt.Errorf("port %d held by %s", port, cur)
	}
	s.used[port] = owner
	return nil
}

func (s *Set) Free(port int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.used[port]; !ok {
		return false
	}
	delete(s.used, port)
	return true
}

func (s *Set) FreeOwner(owner string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for p, o := range s.used {
		if o == owner {
			delete(s.used, p)
			n++
		}
	}
	return n
}

func (s *Set) Owner(port int) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.used[port]
}

func (s *Set) InUse(port int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.used[port]
	return ok
}

func (s *Set) Available() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.max - s.min + 1 - len(s.used) - len(s.reserved)
}

func (s *Set) Range() (int, int) { return s.min, s.max }
