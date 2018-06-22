package balancer

import (
	"fmt"
	"strings"
	"sync"
)

type Stats struct {
	sync.RWMutex
	StatusCode map[string]map[int]uint64
}

func NewStats() *Stats {
	return &Stats{
		StatusCode: map[string]map[int]uint64{},
	}
}

func (s *Stats) Inc(addr string, code int) {
	s.Lock()
	defer s.Unlock()

	if _, ok := s.StatusCode[addr]; !ok {
		s.StatusCode[addr] = map[int]uint64{}
	}
	if _, ok := s.StatusCode[addr][code]; !ok {
		s.StatusCode[addr][code] = 0
	}
	s.StatusCode[addr][code] += 1
}

func (s *Stats) String() string {
	s.RLock()
	defer s.RUnlock()

	result := []string{}
	for addr, sc := range s.StatusCode {
		row := []string{addr}
		for code, count := range sc {
			row = append(row, fmt.Sprintf("%d:%d", code, count))
		}
		result = append(result, strings.Join(row, ", "))
	}

	return strings.Join(result, "\n")
}
