package claps

import (
	"context"
	"fmt"
	"net"
	"strings"
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
	Options

	conn                 net.Conn
	bufInfo, bufPingPong []byte
}

// Connect to NATS server
func (c *NatsConn) Connect(ctx context.Context) {
	c.debug("(%d): Connecting to server: %s", c.ID, c.Address)

	c.bufInfo = make([]byte, rbSizeInfo)
	c.bufPingPong = make([]byte, rbSizePingPong)

	wait := make(chan struct{})
	for {
		go func() {
			defer func() {
				wait <- struct{}{}
			}()

			if c.conn == nil {
				err := c.connect(c.Address, c.Timeout)
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
				c.debug("(%d): Connected to server: %s, addr: %s", c.ID, c.Address, c.conn.LocalAddr())
				c.debug("(%d): Start reading...", c.ID)
			}

			op, args, err := c.read(c.bufPingPong)
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

			c.debug("(%d): %s %s\n", c.ID, op, args)
			if op == "PING" {
				c.conn.Write([]byte(pongCommand))
			}
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
