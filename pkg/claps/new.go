package claps

import (
	"errors"
)

// Errors
var (
	ErrUnknownClient = errors.New("claps: unknown client")
)

const (
	clientNats = "nats"

	defaultBufSize int = 32768
)

// Claps provides
type Claps interface {
	Init(number int)
	Wait()
}

// New Claps
func New(client, address string) (Claps, error) {
	switch client {
	case clientNats:
		return &natsClaps{
			network: "tcp",
			address: address,
		}, nil
	}

	return nil, ErrUnknownClient
}
