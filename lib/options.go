package claps

import (
	"log"
	"math/rand"
	"net"
	"time"
)

type Options struct {
	ID      int
	Address string
	Timeout time.Duration
	Jitter  time.Duration
	Debug   bool
}

func (o *Options) log(format string, v ...interface{}) {
	log.Printf(format, v...)
}

func (o *Options) debug(format string, v ...interface{}) {
	if o.Debug {
		o.log(format, v...)
	}
}

func (o *Options) jitter() {
	t := time.Duration(o.Jitter.Seconds()*rand.Float64()*1000) * time.Millisecond
	log.Printf("(%d): Wait %s and reconnect", o.ID, t)
	time.Sleep(t)
}

type connector interface {
	Close() error
	LocalAddr() net.Addr
}

func (o *Options) disconnect(conn connector) {
	o.debug("(%d): Closing the connection: %s", o.ID, conn.LocalAddr())
	if err := conn.Close(); err != nil {
		o.log("(%d): Can't close the connection, reason=%s", o.ID, err)
	}
	o.debug("(%d): Disconnected", o.ID)
}
