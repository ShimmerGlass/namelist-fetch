package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	err := run()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

func run() error {
	err := loadConfig()
	if err != nil {
		return err
	}

	startServer()

	tick := time.Tick(cfgInterval)
	for ; ; <-tick {
		reloadLists()
	}
}

func startServer() {
	if cfgListenAddr == "" {
		return
	}

	http.Handle("/metrics", promhttp.Handler())
	go func() {
		err := http.ListenAndServe(cfgListenAddr, nil)
		if err != nil {
			slog.Error(err.Error())
			os.Exit(1)
		}
	}()
}
