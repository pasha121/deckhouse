package utils

import (
	"sync"
)

type IPStore struct {
	ips map[string]struct{}
	mu  *sync.Mutex
}

func NewIPStore() *IPStore {
	var mutex sync.Mutex
	return &IPStore{
		mu:  &mutex,
		ips: make(map[string]struct{}),
	}
}

func (s *IPStore) Add(ip string) {
	s.mu.Lock()
	s.ips[ip] = struct{}{}
	s.mu.Unlock()
}

func (s *IPStore) Del(ip string) {
	s.mu.Lock()
	delete(s.ips, ip)
	s.mu.Unlock()
}

func (s *IPStore) IsAllocated(ip string) bool {
	s.mu.Lock()
	_, ok := s.ips[ip]
	s.mu.Unlock()
	if ok {
		return true
	}
	return false
}
