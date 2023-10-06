package data

import (
	"github.com/nats-io/nats.go"
)

type NatsClient struct {
	*nats.EncodedConn
}

// NewNatsClient .
func NewNatsClient(c *Config) (*NatsClient, func(), error) {
	nc, err := nats.Connect(c.Bootstrap.Nats)
	if err != nil {
		return nil, nil, err
	}

	ec, err := nats.NewEncodedConn(nc, nats.JSON_ENCODER)
	if err != nil {
		nc.Close()
		return nil, nil, err
	}

	cleanup := func() {
		ec.Close()
		nc.Close()
	}

	return &NatsClient{
		EncodedConn: ec,
	}, cleanup, nil
}
