package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

type option struct {
	prefix     string
	addr       string
	pool       string
	etcdServer string
	ttl        int64
	hbi        time.Duration
	tbe        time.Duration
}

func newOption() *option {
	var opt option
	flag.StringVar(&opt.prefix, "prefix", "/golb-cluster-1", "etcd key prefix")
	flag.StringVar(&opt.addr, "addr", "127.0.0.1:50001", "serving ip:port address")
	flag.StringVar(&opt.pool, "pool", "web", "pool name")
	flag.StringVar(&opt.etcdServer, "etcd_server", "http://127.0.0.1:2379", "register etcd address")
	flag.Int64Var(&opt.ttl, "ttl", 15, "time to live, it should be greater than heartbeat_interval")
	flag.DurationVar(&opt.hbi, "hbi", time.Second*10, "heartbeat interval")
	flag.DurationVar(&opt.tbe, "tbe", time.Second*3, "timeout before exit")
	flag.Parse()
	return &opt
}

func main() {
	opt := newOption()

	if err := Register(opt.prefix, opt.pool, opt.addr, opt.etcdServer, opt.hbi, opt.ttl); err != nil {
		panic(err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Infof("%s: request from %s, url is %s", opt.addr, r.RemoteAddr, r.URL)
		io.WriteString(w, fmt.Sprintf("hello from %s", opt.addr))
	})

	srv := &http.Server{Addr: opt.addr}
	go func() {
		log.Infof("starting service at %s", opt.addr)
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	gracefulShutdown(srv, opt.tbe)
}

func gracefulShutdown(srv *http.Server, tbe time.Duration) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL, syscall.SIGHUP, syscall.SIGQUIT)

	s := <-ch
	log.Infof("receive signal %v", s)
	UnRegister()

	// do not accept new connection
	// waiting until this server has handled all existing connection
	// API gateway should be responsible for failover (retry work)
	log.Infof("wait %v before exiting...", tbe)
	time.Sleep(tbe)
	if err := srv.Shutdown(nil); err != nil {
		log.Fatal(err)
	}

	log.Infof("gracefully shutdown.")
}
