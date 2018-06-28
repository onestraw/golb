package stats

import (
	"fmt"
	"sort"
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

	keys := []string{}
	for key, _ := range s.StatusCode {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	result := []string{}
	for _, addr := range keys {
		peer := s.StatusCode[addr]
		row := []string{}
		for code, count := range peer {
			row = append(row, fmt.Sprintf("%d:%d", code, count))
		}
		sort.Strings(row)
		row = append([]string{addr}, row...)
		result = append(result, strings.Join(row, ", "))
	}

	return strings.Join(result, "\n")
}

func (s *Stats) Remove(key string) {
	s.Lock()
	defer s.Unlock()
	delete(s.StatusCode, key)
}
