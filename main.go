package main

import (
	"context"
	"flag"
	"fmt"
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
	server     = "0.0.0.0:4222"
	numConns   = 4
	timeout    = 2 * time.Second
	debug      = false
	quiet      = false
	jitter     = 1 * time.Second
	serverType = typeNats
)

const (
	typeCent = "cent"
	typeNats = "nats"
	typeWs   = "ws"
)

func init() {
	flag.StringVar(&server, "s", server, "Server to connect")
	flag.IntVar(&numConns, "nc", numConns, "Number of connections")
	flag.DurationVar(&timeout, "timeout", timeout, "Connect timeout")
	flag.BoolVar(&debug, "debug", debug, "Debug mode, enable for verbose logging (default false)")
	flag.BoolVar(&quiet, "quiet", quiet, "Quiet mode, enable to log nothing (default false)")
	flag.DurationVar(&jitter, "jitter", jitter, "Jitter duration")
	flag.StringVar(&serverType, "t", serverType, "Server type")
}

func main() {
	flag.Parse()

	if quiet {
		log.SetOutput(ioutil.Discard)
		log.SetFlags(0)
	}

	ctx, stop := context.WithCancel(context.Background())
	conns := make([]clap, numConns)
	wg := sync.WaitGroup{}
	wg.Add(numConns)
	for i := 0; i < numConns; i++ {
		conns[i] = newClap(i)
		go func(c clap) {
			c.Connect(ctx)
			wg.Done()
		}(conns[i])
	}

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM)
	<-exit

	stop()
	wg.Wait()
}

type clap interface {
	Connect(context.Context)
}

func newClap(ID int) clap {
	address := server
	if u, err := url.Parse(server); err == nil {
		address = u.Host
	}

	switch serverType {
	case typeNats:
		return &claps.NatsConn{
			ID:      ID,
			Address: address,
			Timeout: timeout,
			Debug:   debug,
			Jitter:  jitter,
		}
	case typeCent:
		return &claps.CentConn{
			ID:      ID,
			URL:     server,
			Debug:   debug,
			Timeout: timeout,
		}
	case typeWs:
		return &claps.WsConn{
			ID:      ID,
			URL:     fmt.Sprintf("%s/%d", server, ID),
			Debug:   debug,
			Timeout: timeout,
			Jitter:  jitter,
		}
	}

	return nil
}
