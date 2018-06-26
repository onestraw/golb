package roundrobin

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// Peer represents a backend server
type Peer struct {
	addr             string
	weight           int
	effective_weight int
	current_weight   int
	down             bool
	sync.RWMutex
}

func (p *Peer) String() string {
	p.RLock()
	defer p.RUnlock()
	return fmt.Sprintf("%s: (w=%d, ew=%d, cw=%d)",
		p.addr, p.weight, p.effective_weight, p.current_weight)
}

func CreatePeer(addr string, weight int) *Peer {
	return &Peer{
		addr:             addr,
		weight:           weight,
		effective_weight: weight,
		current_weight:   0,
		down:             false,
	}
}

// Pool is a group of Peers, one Peer can not belong to multiple Pool
type Pool struct {
	peers   []*Peer
	current uint64
	downNum int
	sync.RWMutex
}

func (p *Pool) String() string {
	p.RLock()
	defer p.RUnlock()
	return fmt.Sprintf("%v", p.peers)
}

func (p *Pool) Size() int {
	return len(p.peers)
}

func (p *Pool) Add(addr string, args ...interface{}) {
	if addr == "" {
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
		p.downNum += 1
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
				p.downNum += 1
			} else {
				p.downNum -= 1
			}
			p.Unlock()

			peer.Lock()
			peer.down = isDown
			peer.Unlock()
		}
	}
}

func (p *Pool) DownPeer(addr string) {
	p.setPeerStatus(addr, true)
}

func (p *Pool) UpPeer(addr string) {
	p.setPeerStatus(addr, false)
}

func (p *Pool) Remove(addr string) {
	if addr == "" {
		return
	}
	p.Lock()
	defer p.Unlock()

	idx := p.indexOfPeer(addr)
	if idx >= 0 && idx < p.Size() {
		if p.peers[idx].down {
			p.downNum -= 1
		}
		p.peers = append(p.peers[:idx], p.peers[idx+1:]...)
	}
}

// GetPeer return peer in smooth weighted roundrobin method
func (p *Pool) Get(args ...interface{}) string {
	p.RLock()
	defer p.RUnlock()

	var best *Peer = nil
	total := 0
	for _, peer := range p.peers {
		if peer.down {
			continue
		}
		peer.Lock()

		total += peer.effective_weight
		peer.current_weight += peer.effective_weight

		if peer.effective_weight < peer.weight {
			peer.effective_weight += 1
		}

		if best == nil || best.current_weight < peer.current_weight {
			best = peer
		}
		peer.Unlock()
	}
	if best != nil {
		best.Lock()
		best.current_weight -= total
		best.Unlock()
		return best.addr
	}
	return ""
}

// EqualGetPeer get peer by turn, without considering weight
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
