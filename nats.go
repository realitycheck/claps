package claps

import (
	"bufio"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	defaultBufSize int = 32768
	defaultTimeout     = 2 * time.Second
	// DefaultMaxReconnect     = 60
	// DefaultReconnectWait    = 2 * time.Second
	// DefaultTimeout          = 2 * time.Second
	// DefaultPingInterval     = 2 * time.Minute
	// DefaultMaxPingOut       = 2

	connectString = "CONNECT {\"name\": \"%s\", \"lang\": \"go\", \"verbose\": false}\r\n"
	pingString    = "PING\r\n"
	pongString    = "PONG\r\n"
)

type natsClaps struct {
	network string
	address string

	// consider mutex
	conns      []*natsConn
	lastConnId int
}

type natsConn struct {
	cc *natsClaps
	id int

	conn net.Conn

	bw *bufio.Writer
}

func (c *natsClaps) Add(number int) {
	wg := sync.WaitGroup{}
	for i := 0; i < number; i++ {
		nc := c.addConn()

		wg.Add(1)
		go func() {
			defer wg.Done()

			var err error

		reconnect:
			if err != nil {
				fmt.Printf("(%d): Can't connect to server: %s, reason: %s\n", nc.id, nc.cc.address, err)

				if nc.conn != nil {
					if err := nc.conn.Close(); err != nil {
						fmt.Printf("(%d): Can't close the connection, reason: %s\n", nc.id, err)
					}
				}
				jitter := time.Duration(1000*rand.Float32()) * time.Millisecond
				fmt.Printf("(%d): Wait %s and reconnect\n", nc.id, jitter)
				time.Sleep(jitter)
			}

			err = nc.createConn()
			if err != nil {
				goto reconnect
			}

			err = nc.sendConnect()
			if err != nil {
				goto reconnect
			}

			go nc.readLoop()
		}()
	}
	wg.Wait()
}

func (c *natsClaps) Done() {

}

func (c *natsClaps) addConn() *natsConn {
	c.lastConnId++

	nc := &natsConn{
		id: c.lastConnId,
		cc: c,
	}
	c.conns = append(c.conns, nc) // consider more efficent growing
	return nc
}

func (nc *natsConn) createConn() error {
	conn, err := net.Dial(nc.cc.network, nc.cc.address)
	if nc.conn = conn; err != nil {
		return err
	}

	if nc.bw != nil {
		nc.bw.Flush()
	}

	nc.bw = bufio.NewWriterSize(nc.conn, defaultBufSize)
	return nil
}

func (nc *natsConn) sendConnect() error {
	nc.conn.SetDeadline(time.Now().Add(defaultTimeout))
	defer nc.conn.SetDeadline(time.Time{})

	c, err := nc.readCmd()
	if err != nil {
		return err
	}

	// The nats protocol should send INFO first always.
	if c.op != "INFO" {
		return fmt.Errorf("nats: expected '%s', got '%s'", "INFO", c.op)
	}

	_, err = nc.bw.WriteString(fmt.Sprintf(connectString, nc.conn.LocalAddr()))
	if err != nil {
		return err
	}

	_, err = nc.bw.WriteString(pingString)
	if err != nil {
		return err
	}

	err = nc.bw.Flush()
	if err != nil {
		return err
	}

	c, err = nc.readCmd()
	if err != nil {
		return err
	}

	if c.op != "PONG" {
		return fmt.Errorf("nats: expected '%s', got '%s'", "PONG", c.op)
	}

	return nil
}

type command struct {
	op, args string
}

// Read a control line and process the intended op.
func (nc *natsConn) readCmd() (command, error) {
	c := command{}

	br := bufio.NewReaderSize(nc.conn, defaultBufSize)
	line, err := br.ReadString('\n')
	if err != nil {
		return c, err
	}

	ss := strings.SplitN(line, " ", 2)
	if len(ss) == 1 {
		c.op = strings.TrimSpace(ss[0])
	} else if len(ss) == 2 {
		c.op, c.args = strings.TrimSpace(ss[0]), strings.TrimSpace(ss[1])
	}

	fmt.Printf("(%d): %v\n", nc.id, c)
	return c, nil
}

func (nc *natsConn) readLoop() {
	for {
		c, err := nc.readCmd()
		if err != nil {
			fmt.Printf("(%d): Can't read command, reason=%s\n", nc.id, err)
		}
		if c.op == "PING" {
			nc.bw.WriteString(pongString)
			nc.bw.Flush()
		}
	}
}
