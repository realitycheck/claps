package claps

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"net"
	"strings"
	"time"
)

const (
	natsNetwork   = "tcp"
	connectString = "CONNECT {\"verbose\": false}\r\n"
	pingString    = "PING\r\n"
	pongString    = "PONG\r\n"
)

type NatsConn struct {
	conn net.Conn
	bw   *bufio.Writer
}

func (nc *NatsConn) Connect(id int, address string, rbSize, wbSize int, connectTimeout time.Duration) {
reconnect:
	err := nc.connect(address, rbSize, wbSize, connectTimeout)
	if err != nil {
		log.Printf("(%d): Can't connect to server: %s, reason: %s\n", id, address, err)

		jitter := time.Duration(1000*rand.Float32()) * time.Millisecond
		log.Printf("(%d): Wait %s and reconnect\n", id, jitter)
		time.Sleep(jitter)

		goto reconnect
	}
	log.Printf("(%d): Connected to server: %s, client: %s\n", id, address, nc.conn.LocalAddr())

	go func() {
		defer func() {
			log.Printf("(%d): Stop reading.\n", id)
			nc.Connect(id, address, rbSize, wbSize, connectTimeout)
		}()

		log.Printf("(%d): Start reading...\n", id)
		for {
			op, args, err := nc.read(rbSize)
			if err != nil {
				log.Printf("(%d): Can't read from server, reason=%s\n", id, err)
				break
			}

			log.Printf("(%d): %s %s\n", id, op, args)

			if op == "PING" {
				nc.bw.WriteString(pongString)
				nc.bw.Flush()
			}
		}
	}()
}

func (nc *NatsConn) connect(address string, rbSize, wbSize int, connectTimeout time.Duration) error {
	if nc.conn != nil {
		if err := nc.conn.Close(); err != nil {
			return err
		}
	}
	conn, err := net.Dial(natsNetwork, address)
	if nc.conn = conn; err != nil {
		return err
	}

	if nc.bw != nil {
		nc.bw.Flush()
	}
	nc.bw = bufio.NewWriterSize(nc.conn, wbSize)

	nc.conn.SetDeadline(time.Now().Add(connectTimeout))
	defer nc.conn.SetDeadline(time.Time{})

	op, _, err := nc.read(rbSize)
	if err != nil {
		return err
	}

	// The nats protocol should send INFO first always.
	if op != "INFO" {
		return fmt.Errorf("nats: expected '%s', got '%s'", "INFO", op)
	}

	nc.bw.WriteString(connectString)
	nc.bw.WriteString(pingString)
	nc.bw.Flush()

	op, _, err = nc.read(rbSize)
	if err != nil {
		return err
	}

	if op != "PONG" {
		return fmt.Errorf("nats: expected '%s', got '%s'", "PONG", op)
	}

	return nil
}

func (nc *NatsConn) read(rbSize int) (string, string, error) {
	var op, args string

	br := bufio.NewReaderSize(nc.conn, rbSize)
	line, err := br.ReadString('\n')
	if err != nil {
		return "", "", err
	}

	ss := strings.SplitN(line, " ", 2)
	if len(ss) == 1 {
		op = strings.TrimSpace(ss[0])
	} else if len(ss) == 2 {
		op, args = strings.TrimSpace(ss[0]), strings.TrimSpace(ss[1])
	}
	return op, args, nil
}
