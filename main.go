package main

import (
	"flag"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/realitycheck/claps/lib"
)

var (
	server   = "0.0.0.0:4222"
	numConns = 4
	rbuf     = 32768
	wbuf     = 32768
	timeout  = 2 * time.Second
	debug    = false
)

func init() {
	flag.StringVar(&server, "s", server, "Server to connect")
	flag.IntVar(&numConns, "nc", numConns, "Number of connections")
	flag.IntVar(&rbuf, "rb", rbuf, "Size of read buffer")
	flag.IntVar(&wbuf, "wb", wbuf, "Size of write buffer")
	flag.DurationVar(&timeout, "timeout", timeout, "Connect timeout")
	flag.BoolVar(&debug, "debug", debug, "Debug mode")
}

func main() {
	flag.Parse()

	address := server
	if u, err := url.Parse(server); err == nil {
		address = u.Host
	}

	conns := make([]*claps.NatsConn, numConns)
	wg := sync.WaitGroup{}
	wg.Add(numConns)
	for i := 0; i < numConns; i++ {
		conns[i] = &claps.NatsConn{}
		go func(i int) {
			conns[i].Connect(i, address, rbuf, wbuf, timeout, debug)
			wg.Done()
		}(i)
	}
	wg.Wait()

	wait := make(chan os.Signal, 1)
	signal.Notify(wait, syscall.SIGINT, syscall.SIGTERM)
	<-wait
}
