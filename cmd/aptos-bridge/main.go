package main

import (
	"os"
	"log"
	"time"
	"context"
	"os/signal"

	"bridge/aptos"
)

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	monitor := aptos.NewAptosMonitor("http://0.0.0.0:8080", 10 * time.Second)
	err := monitor.Init()
	if err != nil {
		log.Println(err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		defer close(done)

		log.Println("aptos bridge is running, press ctrl-c to exit")
		err := monitor.Start(ctx)
		log.Println("aptos done")
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
			log.Println("interrupt")
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