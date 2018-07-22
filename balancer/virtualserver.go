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

const (
	LB_ROUNDROBIN    = "round-robin"
	LB_COSISTENTHASH = "consistent-hash"
	PROTO_HTTP       = "http"
	PROTO_HTTPS      = "https"
	PROTO_GRPC       = "grpc"
	STATUS_ENABLED   = "running"
	STATUS_DISABLED  = "stopped"

	DEFAULT_SERVERNAME  = "localhost"
	DEFAULT_FAILTIMEOUT = 7
	DEFAULT_MAXFAILS    = 2
)

type Pooler interface {
	String() string
	Size() int
	Get(args ...interface{}) string
	Add(addr string, args ...interface{})
	Remove(addr string)
	DownPeer(addr string)
	UpPeer(addr string)
}

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
	pool_lock sync.RWMutex

	retry bool

	ReverseProxy map[string]*httputil.ReverseProxy
	rp_lock      sync.RWMutex

	ServerStats map[string]*stats.Stats
	ss_lock     sync.RWMutex

	server *http.Server
	status string
}

type VirtualServerOption func(*VirtualServer) error

func NameOpt(name string) VirtualServerOption {
	return func(vs *VirtualServer) error {
		if name == "" {
			return ErrVirtualServerNameEmpty
		}
		vs.Name = name
		return nil
	}
}

func AddressOpt(addr string) VirtualServerOption {
	return func(vs *VirtualServer) error {
		if addr == "" {
			return ErrVirtualServerAddressEmpty
		}
		vs.Address = addr
		return nil
	}
}

func ServerNameOpt(serverName string) VirtualServerOption {
	return func(vs *VirtualServer) error {
		if serverName == "" {
			serverName = DEFAULT_SERVERNAME
		}
		vs.ServerName = serverName
		return nil
	}
}

func ProtocolOpt(proto string) VirtualServerOption {
	return func(vs *VirtualServer) error {
		if proto == "" {
			proto = PROTO_HTTP
		}
		if proto != PROTO_HTTP && proto != PROTO_HTTPS {
			return ErrNotSupportedProto
		}
		vs.Protocol = proto
		return nil
	}
}

// TLSOpt should be called after ProtocolOpt
func TLSOpt(certFile, keyFile string) VirtualServerOption {
	return func(vs *VirtualServer) error {
		if vs.Protocol != PROTO_HTTPS {
			return nil
		}
		if _, err := os.Stat(certFile); err != nil {
			return fmt.Errorf("Cert file '%s' does not exist", certFile)
		}
		if _, err := os.Stat(keyFile); err != nil {
			return fmt.Errorf("Key file '%s' does not exist", keyFile)
		}

		vs.CertFile = certFile
		vs.KeyFile = keyFile
		return nil
	}
}

func LBMethodOpt(method string) VirtualServerOption {
	return func(vs *VirtualServer) error {
		if method == "" {
			method = LB_ROUNDROBIN
		}
		if method != LB_ROUNDROBIN && method != LB_COSISTENTHASH {
			return ErrNotSupportedMethod
		}
		vs.LBMethod = method
		return nil
	}
}

func PoolOpt(peers []config.Server) VirtualServerOption {
	return func(vs *VirtualServer) error {
		method := vs.LBMethod
		if method == LB_ROUNDROBIN {
			pairs := make(map[string]int)
			for _, peer := range peers {
				pairs[peer.Address] = peer.Weight
			}
			vs.Pool = roundrobin.CreatePool(pairs)
		} else if method == LB_COSISTENTHASH {
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

func RetryOpt(enable bool) VirtualServerOption {
	return func(vs *VirtualServer) error {
		vs.retry = enable
		return nil
	}
}

func NewVirtualServer(opts ...VirtualServerOption) (*VirtualServer, error) {
	vs := &VirtualServer{
		Protocol:     PROTO_HTTP,
		ServerName:   DEFAULT_SERVERNAME,
		LBMethod:     LB_ROUNDROBIN,
		MaxFails:     DEFAULT_MAXFAILS,
		FailTimeout:  DEFAULT_FAILTIMEOUT,
		retry:        false,
		fails:        make(map[string]int),
		timeout:      make(map[string]int64),
		ReverseProxy: make(map[string]*httputil.ReverseProxy),
		ServerStats:  make(map[string]*stats.Stats),
		status:       STATUS_DISABLED,
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
	s.rp_lock.RLock()
	rp, ok := s.ReverseProxy[peer]
	s.rp_lock.RUnlock()
	if !ok {
		if !strings.HasPrefix(peer, "http://") {
			peer = "http://" + peer
		}
		target, err := url.Parse(peer)
		if err != nil {
			return nil, err
		}
		rp = httputil.NewSingleHostReverseProxy(target)
		s.rp_lock.Lock()
		s.ReverseProxy[peer] = rp
		s.rp_lock.Unlock()
	}
	return rp, nil
}

// fail mark the peer down temporarily if the peer fails MaxFails
func (s *VirtualServer) fail(peer string) {
	s.pool_lock.Lock()
	defer s.pool_lock.Unlock()

	s.fails[peer]++
	if s.fails[peer] >= s.MaxFails {
		log.Infof("Mark down peer: %s", peer)
		s.Pool.DownPeer(peer)
		s.timeout[peer] = time.Now().Unix()
	}
}

// recovery mark the peer up after FailTimeout
func (s *VirtualServer) recovery() {
	s.pool_lock.Lock()
	defer s.pool_lock.Unlock()

	now := time.Now().Unix()
	for k, v := range s.timeout {
		if s.fails[k] >= s.MaxFails && now-v >= s.FailTimeout {
			log.Infof("Mark up peer: %s", k)
			s.Pool.UpPeer(k)
			s.fails[k] = 0
		}
	}
}

type LBResponseWriter struct {
	http.ResponseWriter
	code  int
	bytes int
}

func (w *LBResponseWriter) Write(data []byte) (int, error) {
	size, err := w.ResponseWriter.Write(data)
	w.bytes += size
	return size, err
}

func (w *LBResponseWriter) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}

// ServeHTTP dispatch the request between backend servers
func (s *VirtualServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	timeBegin := time.Now()
	rw := &LBResponseWriter{w, http.StatusOK, 0}
	peer := ""
	defer func() {
		s.StatsInc(peer, r, rw)
		if peer != "" && rw.code/100 == 5 {
			s.fail(peer)
		}
		cost := time.Now().Sub(timeBegin) / time.Millisecond
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

func (s *VirtualServer) StatsInc(addr string, r *http.Request, w *LBResponseWriter) {
	if addr == "" {
		addr = "Load Balancer Error"
	}
	s.ss_lock.RLock()
	ss, ok := s.ServerStats[addr]
	s.ss_lock.RUnlock()
	if !ok {
		s.ss_lock.Lock()
		ss = stats.New()
		s.ServerStats[addr] = ss
		s.ss_lock.Unlock()
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

func (s *VirtualServer) Stats() string {
	keys := []string{}
	for key, _ := range s.ServerStats {
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

func (s *VirtualServer) AddPeer(addr string, args ...interface{}) {
	s.Pool.Add(addr, args...)
}

func (s *VirtualServer) RemovePeer(addr string) {
	s.pool_lock.Lock()
	delete(s.fails, addr)
	delete(s.timeout, addr)
	s.pool_lock.Unlock()

	s.rp_lock.Lock()
	delete(s.ReverseProxy, addr)
	s.rp_lock.Unlock()

	s.ss_lock.Lock()
	delete(s.ServerStats, addr)
	s.ss_lock.Unlock()

	s.Pool.Remove(addr)
}

func (s *VirtualServer) statusSwitch(status string) {
	s.Lock()
	defer s.Unlock()
	s.status = status
}

func (s *VirtualServer) Status() string {
	s.RLock()
	defer s.RUnlock()
	return s.status
}

func (s *VirtualServer) ListenAndServe() error {
	switch s.Protocol {
	case PROTO_HTTP:
		return s.server.ListenAndServe()
	case PROTO_HTTPS:
		return s.server.ListenAndServeTLS(s.CertFile, s.KeyFile)
	}
	return ErrNotSupportedProto
}

func (s *VirtualServer) Run() error {
	if s.Status() == STATUS_ENABLED {
		return fmt.Errorf("%s is already enabled", s.Name)
	}

	log.Infof("Starting [%s], listen %s, proto %s, method %s, pool %v",
		s.Name, s.Address, s.Protocol, s.LBMethod, s.Pool)
	go func() {
		s.statusSwitch(STATUS_ENABLED)
		err := s.ListenAndServe()
		if err != nil {
			log.Errorf("%s ListenAndServe error=%v", s.Name, err)
		}
	}()

	return nil
}

func (s *VirtualServer) Stop() error {
	if s.Status() == STATUS_DISABLED {
		return fmt.Errorf("%s is already disabled", s.Name)
	}

	log.Infof("Stopping [%s]", s.Name)
	if err := s.server.Shutdown(context.Background()); err != nil {
		return fmt.Errorf("%s Shutdown error=%v", s.Name, err)
	}
	s.statusSwitch(STATUS_DISABLED)
	return nil
}
