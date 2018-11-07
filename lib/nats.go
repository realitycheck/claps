package claps

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"strings"
	"time"
)

const (
	natsNetwork    = "tcp"
	connectCommand = "CONNECT {\"verbose\": false}\r\n"
	pingCommand    = "PING\r\n"
	pongCommand    = "PONG\r\n"
)

type NatsConn struct {
	ID      int
	Address string
	Timeout time.Duration
	Jitter  time.Duration
	Debug   bool

	conn      net.Conn
	bw        *bufio.Writer
	connected bool
}

func (nc *NatsConn) logDebugf(format string, v ...interface{}) {
	if nc.Debug {
		log.Printf(format, v...)
	}
}

func (nc *NatsConn) Connect(ctx context.Context) {
	nc.logDebugf("(%d): Connecting to server: %s", nc.ID, nc.Address)

	c := make(chan error)
	r := make(chan command)
	connect := func() { c <- nc.connect(nc.Address, nc.Timeout) }
	read := func() { r <- nc.read() }

	for {
		if !nc.connected {
			go connect()
		} else {
			go read()
		}

		select {
		case <-ctx.Done():
			if nc.conn != nil {
				nc.logDebugf("(%d): Disconnecting, conn: %s", nc.ID, nc.conn.LocalAddr())
				if err := nc.conn.Close(); err != nil {
					log.Printf("(%d): Can't close the connection, reason=%s", nc.ID, err)
				}
			}
			time.AfterFunc(nc.Timeout, func() {
				close(c)
				close(r)
			})
			<-c
			<-r
			nc.logDebugf("(%d): Disconnected.", nc.ID)
			return
		case c := <-r:
			if c.err != nil {
				log.Printf("(%d): Can't read from server, reason=%s", nc.ID, c.err)
				nc.logDebugf("(%d): Stop reading.", nc.ID)
				nc.connected = false
				break
			}
			nc.logDebugf("(%d): %s %s\n", nc.ID, c.op, c.args)
			if c.op == "PING" {
				nc.bw.WriteString(pongCommand)
				nc.bw.Flush()
			}
		case err := <-c:
			if err == nil {
				nc.logDebugf("(%d): Connected to server: %s, conn: %s", nc.ID, nc.Address, nc.conn.LocalAddr())
				nc.logDebugf("(%d): Start reading...", nc.ID)
				nc.connected = true
			} else {
				log.Printf("(%d): Can't connect to server: %s, reason: %s", nc.ID, nc.Address, err)

				jitter := time.Duration(1000*nc.Jitter.Seconds()*rand.Float64()) * time.Millisecond
				log.Printf("(%d): Wait %s and reconnect", nc.ID, jitter)
				time.Sleep(jitter)
			}
		}
	}
}

func (nc *NatsConn) connect(address string, timeout time.Duration) error {
	if nc.conn != nil {
		if err := nc.conn.Close(); err != nil {
			return err
		}
	}
	conn, err := net.DialTimeout(natsNetwork, address, timeout)
	if nc.conn = conn; err != nil {
		return err
	}

	if nc.bw != nil {
		nc.bw.Flush()
	}
	nc.bw = bufio.NewWriter(nc.conn)

	nc.conn.SetDeadline(time.Now().Add(timeout))
	defer nc.conn.SetDeadline(time.Time{})

	c := nc.read()
	if c.err != nil {
		return c.err
	} else if c.op != "INFO" {
		return fmt.Errorf("nats: expected '%s', got '%s'", "INFO", c.op)
	}

	nc.bw.WriteString(connectCommand)
	nc.bw.WriteString(pingCommand)
	nc.bw.Flush()

	c = nc.read()
	if c.err != nil {
		return c.err
	} else if c.op != "PONG" {
		return fmt.Errorf("nats: expected '%s', got '%s'", "PONG", c.op)
	}

	return nil
}

type command struct {
	op, args string
	err      error
}

func (nc *NatsConn) read() command {
	var op, args string

	line, err := bufio.NewReader(nc.conn).ReadString('\n')
	if err != nil {
		return command{err: err}
	}

	ss := strings.SplitN(line, " ", 2)
	if len(ss) == 1 {
		op = strings.TrimSpace(ss[0])
	} else if len(ss) == 2 {
		op, args = strings.TrimSpace(ss[0]), strings.TrimSpace(ss[1])
	}
	return command{op: op, args: args}
}
