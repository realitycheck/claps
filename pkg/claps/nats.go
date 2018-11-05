package claps

import (
	"net"
	"sync"
)

const (
	_CRLF_  = "\r\n"
	_EMPTY_ = ""
	_SPC_   = " "
	_PUB_P_ = "PUB "
)

const (
	_OK_OP_   = "+OK"
	_ERR_OP_  = "-ERR"
	_PONG_OP_ = "PONG"
	_INFO_OP_ = "INFO"
)

const (
	conProto  = "CONNECT %s" + _CRLF_
	pingProto = "PING" + _CRLF_
	pongProto = "PONG" + _CRLF_
	okProto   = _OK_OP_ + _CRLF_
)

type natsClaps struct {
	network string
	address string

	conns []natsConn
}

type natsConn struct {
	id    int
	claps *natsClaps

	conn net.Conn
	err  error
}

func (c *natsClaps) Add(number int) {
	wg := sync.WaitGroup{}
	for i := 0; i < number; i++ {
		nc := c.addConn()

		wg.Add(1)
		go func() {
			nc.createConn()
			nc.sendConnect()
			go nc.loop()
			wg.Done()
		}()
	}
	wg.Wait()
}

func (nc *natsConn) createConn() {
	nc.conn, nc.err = net.Dial(nc.claps.network, nc.claps.address)
}
