package claps

import (
	"context"
	"log"
	"math/rand"
	"sync"
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
	ID      int
	URL     string
	Timeout time.Duration
	Debug   bool
	Jitter  time.Duration

	conn *websocket.Conn
}

func (c *WsConn) logDebugf(format string, v ...interface{}) {
	if c.Debug {
		log.Printf(format, v...)
	}
}

// Connect to WebSocket server
func (c *WsConn) Connect(ctx context.Context) {
	c.logDebugf("(%d): Connecting to server: %s", c.ID, c.URL)

	ok := make(chan struct{})
	setOk := func() { ok <- struct{}{} }

	mu := &sync.Mutex{}
	done := false
	isDone := func() bool {
		mu.Lock()
		defer mu.Unlock()
		return done
	}
	setDone := func() {
		mu.Lock()
		defer mu.Unlock()
		done = true
	}

	disconnect := func() {
		if c.conn != nil {
			c.logDebugf("(%d): Closing the connection: %s", c.ID, c.conn.LocalAddr())
			if err := c.conn.Close(); err != nil {
				log.Printf("(%d): Can't close the connection, reason=%s", c.ID, err)
			}
			c.logDebugf("(%d): Disconnected", c.ID)
			c.conn = nil
		}
	}

	for {
		go func() {
			defer setOk()

			if c.conn == nil {
				ctx, cancel := context.WithDeadline(ctx, time.Now().Add(c.Timeout))
				defer cancel()
				
				conn, _, err := dialer.DialContext(ctx, c.URL, nil)
				if err != nil {
					if !isDone() {
						log.Printf("(%d): Can't connect to server: %s, reason: %s", c.ID, c.URL, err)
						disconnect()

						jitter := time.Duration(1000*c.Jitter.Seconds()*rand.Float64()) * time.Millisecond
						log.Printf("(%d): Wait %s and reconnect", c.ID, jitter)
						time.Sleep(jitter)
					}
					return
				}
				c.conn = conn
				c.logDebugf("(%d): Connected to server: %s, conn: %s", c.ID, c.URL, c.conn.LocalAddr())
				c.logDebugf("(%d): Start reading...", c.ID)
			}

			c.conn.SetReadLimit(maxMessageSize)

			_, message, err := c.conn.ReadMessage()
			if err != nil {
				if !isDone() {
					log.Printf("(%d): Can't read from server, reason=%s", c.ID, err)
					c.logDebugf("(%d): Stop reading", c.ID)
					disconnect()
				}
				return
			}

			c.logDebugf("(%d): Message received: %s", c.ID, message)
		}()

		select {
		case <-ctx.Done():
			setDone()
			disconnect()
			<-ok
			return
		case <-ok:
		}
	}

}
