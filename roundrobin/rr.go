package roundrobin

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

// Peer represents a backend server.
type Peer struct {
	addr            string
	weight          int
	effectiveWeight int
	currentWeight   int
	down            bool
	sync.RWMutex
}

func (p *Peer) String() string {
	p.RLock()
	defer p.RUnlock()
	return fmt.Sprintf("%s: (w=%d, ew=%d, cw=%d)",
		p.addr, p.weight, p.effectiveWeight, p.currentWeight)
}

// CreatePeer return a Peer object.
func CreatePeer(addr string, weight int) *Peer {
	return &Peer{
		addr:            addr,
		weight:          weight,
		effectiveWeight: weight,
		currentWeight:   0,
		down:            false,
	}
}

// Pool is a group of Peers, one Peer can not belong to multiple Pool.
type Pool struct {
	peers   []*Peer
	current uint64
	downNum int
	sync.RWMutex
}

func (p *Pool) String() string {
	p.RLock()
	defer p.RUnlock()
	result := []string{}
	for _, peer := range p.peers {
		result = append(result, peer.addr)
	}
	sort.Strings(result)
	return strings.Join(result, ", ")
}

// Size return the number of the peer.
func (p *Pool) Size() int {
	return len(p.peers)
}

// Add append a peer to the pool if not exists.
func (p *Pool) Add(addr string, args ...interface{}) {
	if addr == "" {
		return
	}
	if idx := p.indexOfPeer(addr); idx >= 0 {
		return
	}
	weight := 1
	if len(args) > 0 {
		if w, ok := args[0].(int); ok {
			weight = w
		}
	}
	peer := CreatePeer(addr, weight)

	p.Lock()
	defer p.Unlock()

	if peer.down {
		p.downNum++
	}
	p.peers = append(p.peers, peer)
}

func (p *Pool) indexOfPeer(addr string) int {
	for i, peer := range p.peers {
		if peer.addr == addr {
			return i
		}
	}
	return -1
}

func (p *Pool) setPeerStatus(addr string, isDown bool) {
	p.RLock()
	idx := p.indexOfPeer(addr)
	p.RUnlock()
	if idx >= 0 && idx < p.Size() {
		peer := p.peers[idx]
		if peer.down != isDown {
			p.Lock()
			if isDown {
				p.downNum++
			} else {
				p.downNum--
			}
			p.Unlock()

			peer.Lock()
			peer.down = isDown
			peer.Unlock()
		}
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

// Remove removes the peer from the pool.
func (p *Pool) Remove(addr string) {
	if addr == "" {
		return
	}
	p.Lock()
	defer p.Unlock()

	idx := p.indexOfPeer(addr)
	if idx >= 0 && idx < p.Size() {
		if p.peers[idx].down {
			p.downNum--
		}
		p.peers = append(p.peers[:idx], p.peers[idx+1:]...)
	}
}

// Get return peer in smooth weighted roundrobin method.
func (p *Pool) Get(args ...interface{}) string {
	p.RLock()
	defer p.RUnlock()

	var best *Peer
	total := 0
	for _, peer := range p.peers {
		if peer.down {
			continue
		}
		peer.Lock()

		total += peer.effectiveWeight
		peer.currentWeight += peer.effectiveWeight

		if peer.effectiveWeight < peer.weight {
			peer.effectiveWeight++
		}

		if best == nil || best.currentWeight < peer.currentWeight {
			best = peer
		}
		peer.Unlock()
	}
	if best != nil {
		best.Lock()
		best.currentWeight -= total
		best.Unlock()
		return best.addr
	}
	return ""
}

// EqualGet get peer by turn, without considering weight.
func (p *Pool) EqualGet() string {
	p.RLock()
	defer p.RUnlock()

	if p.Size() <= 0 {
		return ""
	}
	if p.downNum >= p.Size() {
		return ""
	}

	old := atomic.AddUint64(&p.current, 1) - 1
	idx := old % uint64(p.Size())

	peer := p.peers[idx]
	if !peer.down {
		return peer.addr
	}
	return p.EqualGet()
}

// CreatePool return a Pool object.
func CreatePool(pairs map[string]int) *Pool {
	pool := &Pool{
		current: 0,
		downNum: 0,
	}
	for addr, weight := range pairs {
		pool.Add(addr, weight)
	}
	return pool
}
