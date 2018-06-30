package stats

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

type Stats struct {
	sync.RWMutex
	StatusCode map[string]uint64
	Method     map[string]uint64
	Path       map[string]uint64
	InBytes    uint64
	OutBytes   uint64
}

func New() *Stats {
	return &Stats{
		StatusCode: map[string]uint64{},
		Method:     map[string]uint64{},
		Path:       map[string]uint64{},
		InBytes:    0,
		OutBytes:   0,
	}
}

type Data struct {
	StatusCode string
	Method     string
	Path       string
	InBytes    uint64
	OutBytes   uint64
}

func (s *Stats) Inc(d *Data) {
	s.Lock()
	defer s.Unlock()

	s.StatusCode[d.StatusCode] += 1
	s.Method[d.Method] += 1
	s.Path[d.Path] += 1
	s.InBytes += d.InBytes
	s.OutBytes += d.OutBytes
}

func sortedMapString(dict map[string]uint64) string {
	keys := []string{}
	for key, _ := range dict {
		keys = append(keys, key)
	}

	sort.Strings(keys)
	result := []string{}
	for _, key := range keys {
		result = append(result, fmt.Sprintf("%s:%d", key, dict[key]))
	}

	return strings.Join(result, ", ")
}

const (
	STATUS   = "status_code"
	METHOD   = "method"
	PATH     = "path"
	INBYTES  = "recv_bytes"
	OUTBYTES = "send_bytes"
)

func (s *Stats) String() string {
	s.RLock()
	defer s.RUnlock()

	toS := func(head string, msg interface{}) string {
		return fmt.Sprintf("%s: %v", head, msg)
	}

	result := []string{
		toS(STATUS, sortedMapString(s.StatusCode)),
		toS(METHOD, sortedMapString(s.Method)),
		toS(PATH, sortedMapString(s.Path)),
		toS(INBYTES, s.InBytes),
		toS(OUTBYTES, s.OutBytes),
	}

	return strings.Join(result, "\n")
}
