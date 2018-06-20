package main

import (
	"flag"

	"github.com/onestraw/golb/balancer"
)

func main() {
	var flagConfig = flag.String("config", "golb.json", "json configuration file")
	flag.Parse()

	s, err := balancer.New(*flagConfig)
	if err != nil {
		panic(err)
	}

	s.Run()
}
