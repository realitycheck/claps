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
	numClaps = 4
	rbuf     = 32768
	wbuf     = 32768
	timeout  = 2 * time.Second
)

func init() {
	flag.StringVar(&server, "s", server, "Server to connect")
	flag.IntVar(&numClaps, "nc", numClaps, "Number of connections")
	flag.IntVar(&rbuf, "rb", rbuf, "Size of read buffer")
	flag.IntVar(&wbuf, "wb", wbuf, "Size of write buffer")
	flag.DurationVar(&timeout, "timeout", timeout, "Connect timeout")
}

func main() {
	flag.Parse()

	address := server
	if u, err := url.Parse(server); err == nil {
		address = u.Host
	}

	var conns []*claps.NatsConn
	wg := sync.WaitGroup{}
	wg.Add(numClaps)
	for i := 0; i < numClaps; i++ {
		nc := &claps.NatsConn{}
		conns = append(conns, nc)
		go func(id int) {
			nc.Connect(id, address, rbuf, wbuf, timeout)
			wg.Done()
		}(i)
	}
	wg.Wait()

	wait := make(chan os.Signal, 1)
	signal.Notify(wait, syscall.SIGINT, syscall.SIGTERM)
	<-wait
}
