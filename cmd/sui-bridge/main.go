package main

import (
	"os"
	"log"
	"time"
	"context"
	"os/signal"

	"bridge/sui"
)

const DevnetUrl = "https://fullnode.devnet.sui.io:443"
const LocalnetUrl = "http://localhost:9000"

const ConfigPath = "./sui/config.json"

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	monitor := sui.NewSuiMonitor(DevnetUrl, 10 * time.Second)
	err := monitor.Init(ConfigPath)
	if err != nil {
		log.Println(err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		defer close(done)

		log.Println("sui bridge is running, press ctrl-c to exit")
		err := monitor.Start(ctx)
		log.Println("sui done")
		if err != nil {
			log.Println(err)
			return
		}
	}()

	for {
		select {
		case <-done:
			return
		case <-interrupt:
			log.Println("interrupted by user")
			cancel()
			time.Sleep(2 * time.Second)
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
	// cancel()
}