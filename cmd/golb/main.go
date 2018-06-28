package main

import (
	"flag"

	"github.com/sirupsen/logrus"

	"github.com/onestraw/golb/service"
)

func init() {
	formatter := &logrus.TextFormatter{
		FullTimestamp: true,
		//DisableColors: true,
	}
	logrus.SetFormatter(formatter)
}

func main() {
	var flagConfig = flag.String("config", "golb.json", "json configuration file")
	flag.Parse()

	s, err := service.New(*flagConfig)
	if err != nil {
		panic(err)
	}

	if err := s.Run(); err != nil {
		panic(err)
	}
}
