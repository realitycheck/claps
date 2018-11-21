package claps

import (
	"context"

	"github.com/centrifugal/centrifuge-go"
)

// CentConn represents a connection to Centrifugo server.
type CentConn struct {
	Options

	cli *centrifuge.Client
}

// Connect to Centrifugo server
func (c *CentConn) Connect(ctx context.Context) {
	c.debug("(%d): Connecting to server: %s", c.ID, c.Address)

	config := centrifuge.DefaultConfig()
	config.HandshakeTimeout = c.Timeout

	c.cli = centrifuge.New(c.Address, config)
	defer c.cli.Close()

	c.cli.OnConnect(c)
	c.cli.OnDisconnect(c)
	c.cli.OnError(c)

	go func() {
		err := c.cli.Connect()
		if err != nil {
			c.log("(%d): Can't connect to server: %s, reason: %s", c.ID, c.Address, err)
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
	c.debug("(%d): Connected to server: %s", c.ID, c.Address)
}

func (c *CentConn) OnDisconnect(cli *centrifuge.Client, e centrifuge.DisconnectEvent) {
	if !e.Reconnect {
		c.debug("(%d): Disconnected", c.ID)
	}
}

func (c *CentConn) OnError(cli *centrifuge.Client, e centrifuge.ErrorEvent) {
	c.log("(%d): %s", c.ID, e.Message)
}
