package nats

import (
	"log"

	"github.com/nats-io/nats.go"
)

type JetStreamClient struct {
	nc *nats.Conn
	js nats.JetStreamContext
}

func InitJetStream(url string) (*JetStreamClient, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, err
	}

	js, err := nc.JetStream()
	if err != nil {
		return nil, err
	}

	// Ensure stream exists
	streamName := "BANKING"
	subject := "banking.account.created"
	
	_, err = js.StreamInfo(streamName)
	if err != nil {
		// Stream doesn't exist, create it
		log.Printf("Creating JetStream Stream: %s", streamName)
		_, err = js.AddStream(&nats.StreamConfig{
			Name:     streamName,
			Subjects: []string{subject},
		})
		if err != nil {
			return nil, err
		}
	}

	return &JetStreamClient{
		nc: nc,
		js: js,
	}, nil
}

func (c *JetStreamClient) Publish(subject string, data []byte) error {
	// Async publish can be used, but for exact reliability in outbox sync publish is safer,
	// or we can use js.PublishAsync with wait. We'll use sync for simplicity and strong guarantees.
	_, err := c.js.Publish(subject, data)
	return err
}

func (c *JetStreamClient) Close() {
	c.nc.Close()
}
