package chash

import (
	"fmt"
	"hash/crc32"
	"sort"
	"strings"
	"sync"
)

type Peer struct {
	addr string
}

type Pool struct {
	sync.RWMutex
	replica      int
	vNodes       map[uint32]*Peer
	sortedHashes []uint32
	nodes        map[string]bool
}

func New() *Pool {
	return &Pool{
		replica:      20,
		vNodes:       map[uint32]*Peer{},
		sortedHashes: []uint32{},
		nodes:        map[string]bool{},
	}
}

func (p *Pool) vKey(name string, idx int) string {
	return fmt.Sprintf("%s#%d", name, idx)
}

func (p *Pool) hash(key string) uint32 {
	h := crc32.NewIEEE()
	h.Write([]byte(key))
	return h.Sum32()
}

func (p *Pool) String() string {
	p.RLock()
	defer p.RUnlock()
	result := []string{}
	for key, _ := range p.nodes {
		result = append(result, key)
	}
	return strings.Join(result, ", ")
}

func (p *Pool) Size() int {
	p.RLock()
	defer p.RUnlock()
	return len(p.sortedHashes) / p.replica
}

func (p *Pool) Add(addr string, args ...interface{}) {
	p.Lock()
	defer p.Unlock()

	if _, ok := p.nodes[addr]; ok {
		return
	}
	p.nodes[addr] = true
	peer := &Peer{addr}

	for i := 0; i < p.replica; i++ {
		h := p.hash(p.vKey(peer.addr, i))
		p.vNodes[h] = peer
		p.sortedHashes = append(p.sortedHashes, h)
	}

	sort.Slice(p.sortedHashes, func(i, j int) bool {
		if p.sortedHashes[i] < p.sortedHashes[j] {
			return true
		}
		return false
	})
}

func (p *Pool) Remove(peerAddr string) {
	p.Lock()
	defer p.Unlock()

	if _, ok := p.nodes[peerAddr]; !ok {
		return
	}

	deleteSortedHashes := func(target uint32) {
		for idx, val := range p.sortedHashes {
			if val == target {
				p.sortedHashes = append(p.sortedHashes[:idx], p.sortedHashes[idx+1:]...)
			}
		}
	}

	for i := 0; i < p.replica; i++ {
		h := p.hash(p.vKey(peerAddr, i))
		delete(p.vNodes, h)
		deleteSortedHashes(h)
	}
}

// Get use a key to map the backend server
// key may be a cookie or request_uri
func (p *Pool) Get(args ...interface{}) string {
	if len(args) == 0 {
		return ""
	}

	key, ok := args[0].(string)
	if !ok {
		return ""
	}

	p.RLock()
	defer p.RUnlock()

	if len(p.vNodes) <= 0 {
		return ""
	}

	h := p.hash(key)
	idx := sort.Search(len(p.sortedHashes), func(i int) bool {
		return p.sortedHashes[i] >= h
	})
	if idx >= len(p.sortedHashes) {
		idx = 0
	}
	return p.vNodes[p.sortedHashes[idx]].addr
}

func CreatePool(addrs []string) *Pool {
	pool := New()
	for _, addr := range addrs {
		pool.Add(addr)
	}

	return pool
}
