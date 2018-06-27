package stats

import (
	"fmt"
	"strings"
	"sync"
)

type Stats struct {
	sync.RWMutex
	StatusCode map[string]map[int]uint64
}

func New() *Stats {
	return &Stats{
		StatusCode: map[string]map[int]uint64{},
	}
}

func (s *Stats) Inc(addr string, code int) {
	s.Lock()
	defer s.Unlock()

	peer, ok := s.StatusCode[addr]
	if !ok {
		peer = map[int]uint64{}
		s.StatusCode[addr] = peer
	}
	peer[code] += 1
}

func (s *Stats) String() string {
	s.RLock()
	defer s.RUnlock()

	result := []string{}
	for addr, peer := range s.StatusCode {
		row := []string{addr}
		for code, count := range peer {
			row = append(row, fmt.Sprintf("%d:%d", code, count))
		}
		result = append(result, strings.Join(row, ", "))
	}

	return strings.Join(result, "\n")
}

func (s *Stats) Remove(key string) {
	s.Lock()
	defer s.Unlock()
	delete(s.StatusCode, key)
}
