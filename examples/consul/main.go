package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/consul/api"
)

// ignore all errors
func main() {
	time.Sleep(time.Second * 3)
	var consulAddr, addr, name, prefix string
	flag.StringVar(&consulAddr, "consul-addr", "127.0.0.1:8500", "consul address")
	flag.StringVar(&addr, "addr", "127.0.0.1:5000", "listening address")
	flag.StringVar(&name, "name", "web", "name of the service")
	flag.StringVar(&prefix, "prefix", "t1,t2", "comma-sep list of host/path prefixes to register")
	flag.Parse()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Serving %s from %s on %s\n", r.RequestURI, name, addr)
	})

	go func() {
		fmt.Printf("Listening on %s serving %s\n", addr, prefix)
		http.ListenAndServe(addr, nil)
	}()

	// register consul health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "OK")
	})

	// get host and port as string/int
	_, portStr, _ := net.SplitHostPort(addr)
	port, _ := strconv.Atoi(portStr)

	// register service with health check
	service := &api.AgentServiceRegistration{
		ID:      name + addr,
		Name:    name,
		Port:    port,
		Address: addr,
		Tags:    strings.Split(prefix, ","),
		Meta:    map[string]string{"weight": "1"},
		Check: &api.AgentServiceCheck{
			HTTP:     "http://" + addr + "/health",
			Interval: "5s",
			Timeout:  "2s",
		},
	}
	cfg := api.DefaultConfig()
	cfg.Address = consulAddr
	client, _ := api.NewClient(cfg)

	if err := client.Agent().ServiceRegister(service); err != nil {
		panic(err)
	}
	defer client.Agent().ServiceDeregister(name + addr)
	fmt.Printf("Registered service %q in consul with tags %q\n", name, prefix)

	// gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, os.Kill)
	<-quit
}
