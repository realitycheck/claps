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

type NatsConn struct {
	ID      int
	Address string
	Timeout time.Duration
	Jitter  time.Duration
	Debug   bool

	conn net.Conn

	bufInfo, bufPingPong []byte
}

func (nc *NatsConn) logDebugf(format string, v ...interface{}) {
	if nc.Debug {
		log.Printf(format, v...)
	}
}

func (nc *NatsConn) Connect(ctx context.Context) {
	nc.logDebugf("(%d): Connecting to server: %s", nc.ID, nc.Address)

	nc.bufInfo = make([]byte, rbSizeInfo)
	nc.bufPingPong = make([]byte, rbSizePingPong)

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
		if nc.conn != nil {
			nc.logDebugf("(%d): Closing the connection: %s", nc.ID, nc.conn.LocalAddr())
			if err := nc.conn.Close(); err != nil {
				log.Printf("(%d): Can't close the connection, reason=%s", nc.ID, err)
			}
			nc.logDebugf("(%d): Disconnected", nc.ID)
			nc.conn = nil
		}
	}

	for {
		go func() {
			defer setOk()

			if nc.conn == nil {
				err := nc.connect(nc.Address, nc.Timeout)
				if err != nil {
					if !isDone() {
						log.Printf("(%d): Can't connect to server: %s, reason: %s", nc.ID, nc.Address, err)
						disconnect()

						jitter := time.Duration(1000*nc.Jitter.Seconds()*rand.Float64()) * time.Millisecond
						log.Printf("(%d): Wait %s and reconnect", nc.ID, jitter)
						time.Sleep(jitter)
					}
					return
				}
				nc.logDebugf("(%d): Connected to server: %s, conn: %s", nc.ID, nc.Address, nc.conn.LocalAddr())
				nc.logDebugf("(%d): Start reading...", nc.ID)
			}

			op, args, err := nc.read(nc.bufPingPong)
			if err != nil {
				if !isDone() {
					log.Printf("(%d): Can't read from server, reason=%s", nc.ID, err)
					nc.logDebugf("(%d): Stop reading", nc.ID)
					disconnect()
				}
				return
			}

			nc.logDebugf("(%d): %s %s\n", nc.ID, op, args)
			if op == "PING" {
				nc.conn.Write([]byte(pongCommand))
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

func (nc *NatsConn) connect(address string, timeout time.Duration) error {
	conn, err := net.DialTimeout(natsNetwork, address, timeout)
	if nc.conn = conn; err != nil {
		return err
	}

	nc.conn.SetDeadline(time.Now().Add(timeout))
	defer nc.conn.SetDeadline(time.Time{})
	op, _, err := nc.read(nc.bufInfo)
	if err != nil {
		return err
	} else if op != "INFO" {
		return fmt.Errorf("nats: expected '%s', got '%s'", "INFO", op)
	}

	nc.conn.Write([]byte(connectCommand + pingCommand))
	op, _, err = nc.read(nc.bufPingPong)
	if err != nil {
		return err
	} else if op != "PONG" {
		return fmt.Errorf("nats: expected '%s', got '%s'", "PONG", op)
	}

	return nil
}

func (nc *NatsConn) read(buf []byte) (string, string, error) {
	var op, args string

	n, err := nc.conn.Read(buf)
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
