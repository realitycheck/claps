package claps

import (
	"context"
	"time"

	"github.com/gorilla/websocket"
)

const (
	maxMessageSize = 256
)

var dialer = &websocket.Dialer{
	ReadBufferSize:  maxMessageSize,
	WriteBufferSize: 8,
}

// WsConn represents a connection to generic WebSocket server.
type WsConn struct {
	Options

	conn *websocket.Conn
}

// Connect to WebSocket server
func (c *WsConn) Connect(ctx context.Context) {
	c.debug("(%d): Connecting to server: %s", c.ID, c.Address)

	wait := make(chan struct{})
	for {
		go func() {
			defer func() {
				wait <- struct{}{}
			}()

			if c.conn == nil {
				dctx, cancel := context.WithDeadline(ctx, time.Now().Add(c.Timeout))
				conn, _, err := dialer.DialContext(dctx, c.Address, nil)
				cancel()
				if err != nil {
					select {
					case <-ctx.Done():
					default:
						c.log("(%d): Can't connect to server: %s, reason: %s", c.ID, c.Address, err)
						if c.conn != nil {
							c.disconnect(c.conn)
							c.conn = nil
						}
						c.jitter()
					}
					return
				}
				c.conn = conn
				c.debug("(%d): Connected to server: %s, addr: %s", c.ID, c.Address, c.conn.LocalAddr())
				c.debug("(%d): Start reading...", c.ID)
			}

			c.conn.SetReadLimit(maxMessageSize)
			_, message, err := c.conn.ReadMessage()
			if err != nil {
				select {
				case <-ctx.Done():
				default:
					c.log("(%d): Can't read from server, reason=%s", c.ID, err)
					c.debug("(%d): Stop reading", c.ID)
					if c.conn != nil {
						c.disconnect(c.conn)
						c.conn = nil
					}
					c.jitter()
				}
				return
			}

			c.debug("(%d): Message received: %s", c.ID, message)
		}()

		select {
		case <-ctx.Done():
			if c.conn != nil {
				c.disconnect(c.conn)
				c.conn = nil
			}
			<-wait
			return
		case <-wait:
		}
	}

}
