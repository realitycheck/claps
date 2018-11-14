package claps

import (
	"context"
	"log"
	"time"

	"github.com/centrifugal/centrifuge-go"
)

// CentConn represents a connection to Centrifugo server.
type CentConn struct {
	ID      int
	URL     string
	Debug   bool
	Timeout time.Duration

	c *centrifuge.Client
}

func (cc *CentConn) logDebugf(format string, v ...interface{}) {
	if cc.Debug {
		log.Printf(format, v...)
	}
}

// Connect to Centrifugo server
func (cc *CentConn) Connect(ctx context.Context) {
	config := centrifuge.DefaultConfig()
	config.ReadTimeout = cc.Timeout
	config.WriteTimeout = cc.Timeout

	cc.c = centrifuge.New(cc.URL, config)
	defer cc.c.Close()

	cc.c.OnConnect(cc)
	cc.c.OnDisconnect(cc)
	cc.c.OnError(cc)

	go func() {
		err := cc.c.Connect()
		if err != nil {
			log.Printf("(%d): Can't connect to server: %s, reason: %s", cc.ID, cc.URL, err)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		}
	}
}

func (cc *CentConn) OnConnect(c *centrifuge.Client, e centrifuge.ConnectEvent) {
	cc.logDebugf("(%d): Connected to server: %s", cc.ID, cc.URL)
}

func (cc *CentConn) OnDisconnect(c *centrifuge.Client, e centrifuge.DisconnectEvent) {
	if !e.Reconnect {
		cc.logDebugf("(%d): Disconnected", cc.ID)
	}
}

func (cc *CentConn) OnError(c *centrifuge.Client, e centrifuge.ErrorEvent) {
	log.Printf("(%d): %s", cc.ID, e.Message)
}
