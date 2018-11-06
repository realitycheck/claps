package claps

const (
	nameNats = "nats"
)

// Claps provides
type Claps interface {
	Add(number int)
	Done()
}

// New Claps
func New(name, address string) Claps {
	switch name {
	case nameNats:
		return &natsClaps{
			network: "tcp",
			address: address,
		}
	}

	panic("claps: unknown name")
}
