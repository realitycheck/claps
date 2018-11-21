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

	cli *centrifuge.Client
}

func (c *CentConn) logDebugf(format string, v ...interface{}) {
	if c.Debug {
		log.Printf(format, v...)
	}
}

// Connect to Centrifugo server
func (c *CentConn) Connect(ctx context.Context) {
	config := centrifuge.DefaultConfig()
	config.ReadTimeout = c.Timeout
	config.WriteTimeout = c.Timeout

	c.cli = centrifuge.New(c.URL, config)
	defer c.cli.Close()

	c.cli.OnConnect(c)
	c.cli.OnDisconnect(c)
	c.cli.OnError(c)

	go func() {
		err := c.cli.Connect()
		if err != nil {
			log.Printf("(%d): Can't connect to server: %s, reason: %s", c.ID, c.URL, err)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		}
	}
}

func (c *CentConn) OnConnect(cli *centrifuge.Client, e centrifuge.ConnectEvent) {
	c.logDebugf("(%d): Connected to server: %s", c.ID, c.URL)
}

func (c *CentConn) OnDisconnect(cli *centrifuge.Client, e centrifuge.DisconnectEvent) {
	if !e.Reconnect {
		c.logDebugf("(%d): Disconnected", c.ID)
	}
}

func (c *CentConn) OnError(cli *centrifuge.Client, e centrifuge.ErrorEvent) {
	log.Printf("(%d): %s", c.ID, e.Message)
}
