package main

import (
	"github.com/realitycheck/claps/pkg/claps"
)

func main() {
	claps := claps.New("nats", "0.0.0.0:4222")

	claps.Add(1)

	claps.Done()
}
