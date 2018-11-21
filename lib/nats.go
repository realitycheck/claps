package claps

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	natsNetwork    = "tcp"
	connectCommand = "CONNECT {\"verbose\": false}\r\n"
	pingCommand    = "PING\r\n"
	pongCommand    = "PONG\r\n"

	rbSizePingPong = 8
	rbSizeInfo     = 256
)

// NatsConn represents a connection to NATS server.
type NatsConn struct {
	ID      int
	Address string
	Timeout time.Duration
	Jitter  time.Duration
	Debug   bool

	conn net.Conn

	bufInfo, bufPingPong []byte
}

func (c *NatsConn) logDebugf(format string, v ...interface{}) {
	if c.Debug {
		log.Printf(format, v...)
	}
}

// Connect to NATS server
func (c *NatsConn) Connect(ctx context.Context) {
	c.logDebugf("(%d): Connecting to server: %s", c.ID, c.Address)

	c.bufInfo = make([]byte, rbSizeInfo)
	c.bufPingPong = make([]byte, rbSizePingPong)

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
				err := c.connect(c.Address, c.Timeout)
				if err != nil {
					if !isDone() {
						log.Printf("(%d): Can't connect to server: %s, reason: %s", c.ID, c.Address, err)
						disconnect()

						jitter := time.Duration(1000*c.Jitter.Seconds()*rand.Float64()) * time.Millisecond
						log.Printf("(%d): Wait %s and reconnect", c.ID, jitter)
						time.Sleep(jitter)
					}
					return
				}
				c.logDebugf("(%d): Connected to server: %s, conn: %s", c.ID, c.Address, c.conn.LocalAddr())
				c.logDebugf("(%d): Start reading...", c.ID)
			}

			op, args, err := c.read(c.bufPingPong)
			if err != nil {
				if !isDone() {
					log.Printf("(%d): Can't read from server, reason=%s", c.ID, err)
					c.logDebugf("(%d): Stop reading", c.ID)
					disconnect()
				}
				return
			}

			c.logDebugf("(%d): %s %s\n", c.ID, op, args)
			if op == "PING" {
				c.conn.Write([]byte(pongCommand))
			}
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

func (c *NatsConn) connect(address string, timeout time.Duration) error {
	conn, err := net.DialTimeout(natsNetwork, address, timeout)
	if c.conn = conn; err != nil {
		return err
	}

	c.conn.SetDeadline(time.Now().Add(timeout))
	defer c.conn.SetDeadline(time.Time{})
	op, _, err := c.read(c.bufInfo)
	if err != nil {
		return err
	} else if op != "INFO" {
		return fmt.Errorf("nats: expected '%s', got '%s'", "INFO", op)
	}

	c.conn.Write([]byte(connectCommand + pingCommand))
	op, _, err = c.read(c.bufPingPong)
	if err != nil {
		return err
	} else if op != "PONG" {
		return fmt.Errorf("nats: expected '%s', got '%s'", "PONG", op)
	}

	return nil
}

func (c *NatsConn) read(buf []byte) (string, string, error) {
	var op, args string

	n, err := c.conn.Read(buf)
	if err != nil {
		return "", "", err
	}

	line := string(buf[:n])
	ss := strings.SplitN(line, " ", 2)
	if len(ss) == 1 {
		op = strings.TrimSpace(ss[0])
	} else if len(ss) == 2 {
		op, args = strings.TrimSpace(ss[0]), strings.TrimSpace(ss[1])
	}
	return op, args, nil
}
