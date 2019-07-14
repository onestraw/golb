package balancer

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/onestraw/golb/chash"
	"github.com/onestraw/golb/config"
	"github.com/onestraw/golb/retry"
	"github.com/onestraw/golb/roundrobin"
	"github.com/onestraw/golb/stats"
)

// constants
const (
	LBRoundRobin     = "round-robin"
	LBConsistentHash = "consistent-hash"
	ProtoHTTP        = "http"
	ProtoHTTPS       = "https"
	ProtoGRPC        = "grpc"
	StatusEnabled    = "running"
	StatusDisabled   = "stopped"

	DefaultServerName  = "localhost"
	DefaultFailTimeout = 7
	DefaultMaxFails    = 2
)

// Pooler is a LB method interface.
type Pooler interface {
	String() string
	Size() int
	Get(args ...interface{}) string
	Add(addr string, args ...interface{})
	Remove(addr string)
	DownPeer(addr string)
	UpPeer(addr string)
}

// VirtualServer defines a LoadBalancer instance.
type VirtualServer struct {
	sync.RWMutex
	Name       string
	Address    string
	ServerName string
	Protocol   string
	CertFile   string
	KeyFile    string
	LBMethod   string
	Pool       Pooler

	// maximum fails before mark peer down
	MaxFails int
	fails    map[string]int

	// timeout before retry a down peer
	FailTimeout int64
	timeout     map[string]int64

	// used for fails/timeout
	poolLock sync.RWMutex

	retry bool

	ReverseProxy map[string]*httputil.ReverseProxy
	rpLock       sync.RWMutex

	ServerStats map[string]*stats.Stats
	ssLock      sync.RWMutex

	server *http.Server
	status string
}

// VirtualServerOption provides option setter for VirtualServer.
type VirtualServerOption func(*VirtualServer) error

// NameOpt returns a function to set name.
func NameOpt(name string) VirtualServerOption {
	return func(vs *VirtualServer) error {
		if name == "" {
			return ErrVirtualServerNameEmpty
		}
		vs.Name = name
		return nil
	}
}

// AddressOpt returns a function to set address.
func AddressOpt(addr string) VirtualServerOption {
	return func(vs *VirtualServer) error {
		if addr == "" {
			return ErrVirtualServerAddressEmpty
		}
		vs.Address = addr
		return nil
	}
}

// ServerNameOpt returns a function to set server name.
func ServerNameOpt(serverName string) VirtualServerOption {
	return func(vs *VirtualServer) error {
		if serverName == "" {
			serverName = DefaultServerName
		}
		vs.ServerName = serverName
		return nil
	}
}

// ProtocolOpt returns a function to set protocol.
func ProtocolOpt(proto string) VirtualServerOption {
	return func(vs *VirtualServer) error {
		if proto == "" {
			proto = ProtoHTTP
		}
		if proto != ProtoHTTP && proto != ProtoHTTPS {
			return ErrNotSupportedProto
		}
		vs.Protocol = proto
		return nil
	}
}

// TLSOpt returns a function to set TLS and should be called after ProtocolOpt.
func TLSOpt(certFile, keyFile string) VirtualServerOption {
	return func(vs *VirtualServer) error {
		if vs.Protocol != ProtoHTTPS {
			return nil
		}
		if _, err := os.Stat(certFile); err != nil {
			return fmt.Errorf("cert file '%s' does not exist", certFile)
		}
		if _, err := os.Stat(keyFile); err != nil {
			return fmt.Errorf("key file '%s' does not exist", keyFile)
		}

		vs.CertFile = certFile
		vs.KeyFile = keyFile
		return nil
	}
}

// LBMethodOpt returns a function to set LBMethod.
func LBMethodOpt(method string) VirtualServerOption {
	return func(vs *VirtualServer) error {
		if method == "" {
			method = LBRoundRobin
		}
		if method != LBRoundRobin && method != LBConsistentHash {
			return ErrNotSupportedMethod
		}
		vs.LBMethod = method
		return nil
	}
}

// PoolOpt returns a function to set pool.
func PoolOpt(peers []config.Server) VirtualServerOption {
	return func(vs *VirtualServer) error {
		method := vs.LBMethod
		if method == LBRoundRobin {
			pairs := make(map[string]int)
			for _, peer := range peers {
				pairs[peer.Address] = peer.Weight
			}
			vs.Pool = roundrobin.CreatePool(pairs)
		} else if method == LBConsistentHash {
			addrs := make([]string, len(peers))
			for i, peer := range peers {
				addrs[i] = peer.Address
			}
			vs.Pool = chash.CreatePool(addrs)
		} else {
			return ErrNotSupportedMethod
		}
		return nil
	}
}

// RetryOpt returns a function to set retry.
func RetryOpt(enable bool) VirtualServerOption {
	return func(vs *VirtualServer) error {
		vs.retry = enable
		return nil
	}
}

// NewVirtualServer returns a VirtualServer object.
func NewVirtualServer(opts ...VirtualServerOption) (*VirtualServer, error) {
	vs := &VirtualServer{
		Protocol:     ProtoHTTP,
		ServerName:   DefaultServerName,
		LBMethod:     LBRoundRobin,
		MaxFails:     DefaultMaxFails,
		FailTimeout:  DefaultFailTimeout,
		retry:        false,
		fails:        make(map[string]int),
		timeout:      make(map[string]int64),
		ReverseProxy: make(map[string]*httputil.ReverseProxy),
		ServerStats:  make(map[string]*stats.Stats),
		status:       StatusDisabled,
	}
	for _, opt := range opts {
		if err := opt(vs); err != nil {
			return nil, err
		}
	}
	if vs.Name == "" {
		return nil, NameOpt("")(vs)
	}
	if vs.Address == "" {
		return nil, AddressOpt("")(vs)
	}
	vs.server = &http.Server{Addr: vs.Address, Handler: vs}
	if vs.retry {
		vs.server.Handler = retry.Retry(vs)
	}

	return vs, nil
}

func (s *VirtualServer) getReverseProxy(peer string) (*httputil.ReverseProxy, error) {
	s.rpLock.RLock()
	rp, ok := s.ReverseProxy[peer]
	s.rpLock.RUnlock()
	if !ok {
		if !strings.HasPrefix(peer, "http://") {
			peer = "http://" + peer
		}
		target, err := url.Parse(peer)
		if err != nil {
			return nil, err
		}
		rp = httputil.NewSingleHostReverseProxy(target)
		s.rpLock.Lock()
		s.ReverseProxy[peer] = rp
		s.rpLock.Unlock()
	}
	return rp, nil
}

// fail mark the peer down temporarily if the peer fails MaxFails.
func (s *VirtualServer) fail(peer string) {
	s.poolLock.Lock()
	defer s.poolLock.Unlock()

	s.fails[peer]++
	if s.fails[peer] >= s.MaxFails {
		log.Infof("Mark down peer: %s", peer)
		s.Pool.DownPeer(peer)
		s.timeout[peer] = time.Now().Unix()
	}
}

// recovery mark the peer up after FailTimeout.
func (s *VirtualServer) recovery() {
	s.poolLock.Lock()
	defer s.poolLock.Unlock()

	now := time.Now().Unix()
	for k, v := range s.timeout {
		if s.fails[k] >= s.MaxFails && now-v >= s.FailTimeout {
			log.Infof("Mark up peer: %s", k)
			s.Pool.UpPeer(k)
			s.fails[k] = 0
		}
	}
}

type lbResponseWriter struct {
	http.ResponseWriter
	code  int
	bytes int
}

func (w *lbResponseWriter) Write(data []byte) (int, error) {
	size, err := w.ResponseWriter.Write(data)
	w.bytes += size
	return size, err
}

func (w *lbResponseWriter) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}

// ServeHTTP dispatch the request between backend servers.
func (s *VirtualServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	timeBegin := time.Now()
	rw := &lbResponseWriter{w, http.StatusOK, 0}
	peer := ""
	defer func() {
		s.StatsInc(peer, r, rw)
		if peer != "" && rw.code/100 == 5 {
			s.fail(peer)
		}
		cost := time.Since(timeBegin) / time.Millisecond
		log.Infof("%s - %s %s(%s)%s %s %dms- %d", r.RemoteAddr, r.Method, r.Host, peer, r.URL, r.Proto, cost, rw.code)
	}()

	s.RLock()
	defer s.RUnlock()

	s.recovery()

	// check the request’s header field "Host"
	if r.Host != s.ServerName {
		log.Errorf("Host not match, host=%s", r.Host)
		WriteError(rw, ErrHostNotMatch)
		return
	}

	// use client's address as hash key if using consistent-hash method
	peer = s.Pool.Get(r.RemoteAddr)
	if peer == "" {
		log.Errorf("Get peer err=%v", ErrPeerNotFound.ErrMsg)
		WriteError(rw, ErrPeerNotFound)
		return
	}

	rp, err := s.getReverseProxy(peer)
	if err != nil {
		log.Errorf("GetReverseProxy err=%v", err)
		WriteError(rw, ErrInternalBalancer)
		return
	}

	rp.ServeHTTP(rw, r)
}

// StatsInc adds a request info.
func (s *VirtualServer) StatsInc(addr string, r *http.Request, w *lbResponseWriter) {
	if addr == "" {
		addr = "Load Balancer Error"
	}
	s.ssLock.RLock()
	ss, ok := s.ServerStats[addr]
	s.ssLock.RUnlock()
	if !ok {
		s.ssLock.Lock()
		ss = stats.New()
		s.ServerStats[addr] = ss
		s.ssLock.Unlock()
	}
	data := &stats.Data{
		StatusCode: strconv.Itoa(w.code),
		Method:     r.Method,
		Path:       r.URL.Path,
		InBytes:    uint64(r.ContentLength),
		OutBytes:   uint64(w.bytes),
	}
	ss.Inc(data)
}

// Stats return the stats info.
func (s *VirtualServer) Stats() string {
	keys := []string{}
	for key := range s.ServerStats {
		keys = append(keys, key)
	}

	sort.Strings(keys)
	result := []string{
		fmt.Sprintf("Pool-%s", s.Name),
	}
	for _, peer := range keys {
		ss := s.ServerStats[peer]
		result = append(result, fmt.Sprintf("%s\n%s\n------", peer, ss))
	}
	return strings.Join(result, "\n")
}

// AddPeer adds one peer to the pool.
func (s *VirtualServer) AddPeer(addr string, args ...interface{}) {
	s.Pool.Add(addr, args...)
}

// RemovePeer removes the peer from the pool and related.
func (s *VirtualServer) RemovePeer(addr string) {
	s.poolLock.Lock()
	delete(s.fails, addr)
	delete(s.timeout, addr)
	s.poolLock.Unlock()

	s.rpLock.Lock()
	delete(s.ReverseProxy, addr)
	s.rpLock.Unlock()

	s.ssLock.Lock()
	delete(s.ServerStats, addr)
	s.ssLock.Unlock()

	s.Pool.Remove(addr)
}

func (s *VirtualServer) statusSwitch(status string) {
	s.Lock()
	defer s.Unlock()
	s.status = status
}

// Status return the server status.
func (s *VirtualServer) Status() string {
	s.RLock()
	defer s.RUnlock()
	return s.status
}

func (s *VirtualServer) listenAndServe() error {
	switch s.Protocol {
	case ProtoHTTP:
		return s.server.ListenAndServe()
	case ProtoHTTPS:
		return s.server.ListenAndServeTLS(s.CertFile, s.KeyFile)
	}
	return ErrNotSupportedProto
}

// Run starts the server.
func (s *VirtualServer) Run() error {
	if s.Status() == StatusEnabled {
		return fmt.Errorf("%s is already enabled", s.Name)
	}

	log.Infof("Starting [%s], listen %s, proto %s, method %s, pool %v",
		s.Name, s.Address, s.Protocol, s.LBMethod, s.Pool)
	go func() {
		s.statusSwitch(StatusEnabled)
		err := s.listenAndServe()
		if err != nil {
			log.Errorf("%s ListenAndServe error=%v", s.Name, err)
		}
	}()

	return nil
}

// Stop stops the server.
func (s *VirtualServer) Stop() error {
	if s.Status() == StatusDisabled {
		return fmt.Errorf("%s is already disabled", s.Name)
	}

	log.Infof("Stopping [%s]", s.Name)
	if err := s.server.Shutdown(context.Background()); err != nil {
		return fmt.Errorf("%s Shutdown error=%v", s.Name, err)
	}
	s.statusSwitch(StatusDisabled)
	return nil
}
