// Package pubsub wraps the Cloud Pub/Sub v2 publisher behind the tiny
// services.Publisher interface (Publish(ctx, attrs, data)). It's used to kick
// off asynchronous in-character email replies.
package pubsub

import (
	"context"

	gpubsub "cloud.google.com/go/pubsub/v2"
)

// Publisher publishes messages to a single topic.
type Publisher struct {
	client *gpubsub.Client
	pub    *gpubsub.Publisher
}

// NewPublisher builds a publisher for the given project + topic. The caller
// should Close it on shutdown.
func NewPublisher(ctx context.Context, projectID, topicID string) (*Publisher, error) {
	client, err := gpubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return &Publisher{client: client, pub: client.Publisher(topicID)}, nil
}

// Publish sends one message and blocks until the server assigns an ID.
func (p *Publisher) Publish(ctx context.Context, attrs map[string]string, data []byte) (string, error) {
	res := p.pub.Publish(ctx, &gpubsub.Message{Data: data, Attributes: attrs})
	return res.Get(ctx)
}

// Close flushes and releases the publisher + client.
func (p *Publisher) Close() error {
	p.pub.Stop()
	return p.client.Close()
}
