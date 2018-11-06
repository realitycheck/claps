package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/realitycheck/claps"
)

var (
	clapsName     string
	serverAddress string
	numClaps      int
)

func main() {
	flag.StringVar(&clapsName, "c", "nats", "")
	flag.StringVar(&serverAddress, "s", "0.0.0.0:4222", "")
	flag.IntVar(&numClaps, "nc", 4, "")

	flag.Parse()

	cc := claps.New(clapsName, serverAddress)
	defer cc.Done()

	cc.Add(numClaps)

	wait(syscall.SIGINT, syscall.SIGTERM)
}

func wait(sig ...os.Signal) {
	wait := make(chan os.Signal, 1)
	signal.Notify(wait, sig...)

	<-wait
}
