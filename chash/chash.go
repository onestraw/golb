package chash

import (
	"fmt"
	"hash/crc32"
	"sort"
	"strings"
	"sync"
)

// Peer defines a single server.
type Peer struct {
	sync.RWMutex
	addr string
	down bool
}

// Pool is a set of Peers.
type Pool struct {
	sync.RWMutex
	replica      int
	vNodes       map[uint32]*Peer
	sortedHashes []uint32
	nodes        map[string]bool
	downNum      int
}

// New returns a Pool object.
func New() *Pool {
	return &Pool{
		replica:      20,
		vNodes:       map[uint32]*Peer{},
		sortedHashes: []uint32{},
		nodes:        map[string]bool{},
		downNum:      0,
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
	for key := range p.nodes {
		result = append(result, key)
	}
	sort.Strings(result)
	return strings.Join(result, ", ")
}

// Size return the number of peers.
func (p *Pool) Size() int {
	p.RLock()
	defer p.RUnlock()
	return len(p.sortedHashes) / p.replica
}

// Add adds a peer by address.
func (p *Pool) Add(addr string, args ...interface{}) {
	p.Lock()
	defer p.Unlock()

	if _, ok := p.nodes[addr]; ok {
		return
	}
	p.nodes[addr] = true
	peer := &Peer{addr: addr, down: false}

	for i := 0; i < p.replica; i++ {
		h := p.hash(p.vKey(peer.addr, i))
		p.vNodes[h] = peer
		p.sortedHashes = append(p.sortedHashes, h)
	}

	sort.Slice(p.sortedHashes, func(i, j int) bool {
		return p.sortedHashes[i] < p.sortedHashes[j]
	})
}

// Remove deletes a peer by address.
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
		if p.vNodes[h].down {
			p.downNum--
		}
		delete(p.vNodes, h)
		deleteSortedHashes(h)
	}
}

func (p *Pool) setPeerStatus(peerAddr string, isDown bool) {
	p.Lock()
	defer p.Unlock()

	if _, ok := p.nodes[peerAddr]; !ok {
		return
	}
	idx := 1
	h := p.hash(p.vKey(peerAddr, idx))
	peer := p.vNodes[h]
	if peer.down != isDown {
		if isDown {
			p.downNum++
		} else {
			p.downNum--
		}
		peer.Lock()
		peer.down = isDown
		peer.Unlock()
	}
}

// DownPeer mark the peer down.
func (p *Pool) DownPeer(addr string) {
	p.setPeerStatus(addr, true)
}

// UpPeer mark the peer up.
func (p *Pool) UpPeer(addr string) {
	p.setPeerStatus(addr, false)
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

	if len(p.vNodes) <= 0 || p.downNum >= p.Size() {
		return ""
	}

	h := p.hash(key)
	idx := sort.Search(len(p.sortedHashes), func(i int) bool {
		return p.sortedHashes[i] >= h && !p.vNodes[p.sortedHashes[i]].down
	})
	if idx >= len(p.sortedHashes) {
		idx = 0
	}
	return p.vNodes[p.sortedHashes[idx]].addr
}

// CreatePool returns a Pool object.
func CreatePool(addrs []string) *Pool {
	pool := New()
	for _, addr := range addrs {
		pool.Add(addr)
	}

	return pool
}
