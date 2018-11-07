package main

import (
	"context"
	"flag"
	"io/ioutil"
	"log"
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
	timeout  = 2 * time.Second
	debug    = false
	quiet    = false
	jitter   = 1 * time.Second
)

func init() {
	flag.StringVar(&server, "s", server, "Server to connect")
	flag.IntVar(&numConns, "nc", numConns, "Number of connections")
	flag.DurationVar(&timeout, "timeout", timeout, "Connect timeout")
	flag.BoolVar(&debug, "debug", debug, "Debug mode, enable for verbose logging (default false)")
	flag.BoolVar(&quiet, "quiet", quiet, "Quiet mode, enable to log nothing (default false)")
	flag.DurationVar(&jitter, "jitter", jitter, "Jitter duration")
}

func main() {
	flag.Parse()

	address := server
	if u, err := url.Parse(server); err == nil {
		address = u.Host
	}

	if quiet {
		log.SetOutput(ioutil.Discard)
		log.SetFlags(0)
	}

	ctx, cancel := context.WithCancel(context.Background())
	conns := make([]*claps.NatsConn, numConns)
	wg := sync.WaitGroup{}
	wg.Add(numConns)
	for i := 0; i < numConns; i++ {
		conns[i] = &claps.NatsConn{
			ID:      i,
			Address: address,
			Timeout: timeout,
			Debug:   debug,
			Jitter:  jitter,
		}
		nc := conns[i]
		go func() {
			nc.Connect(ctx)
			wg.Done()
		}()
	}

	wait := make(chan os.Signal, 1)
	signal.Notify(wait, syscall.SIGINT, syscall.SIGTERM)
	<-wait

	cancel()

	wg.Wait()
}
