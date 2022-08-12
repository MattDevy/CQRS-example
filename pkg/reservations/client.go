package reservations

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
	"google.golang.org/api/option"
)

const (
	// RoomCommandsTopic is the PubSub topic commands are sent/recieved
	RoomCommandsTopic = "rooms.commands"
	// CommandTypeAttributeKey is the Attribute Key that the CommandTypes are sent using
	CommandTypeAttributeKey = "CommandType"
)

// Client is a pubsub client that will send Commands to the command handler server
type Client struct {
	p     *pubsub.Client
	topic *pubsub.Topic
}

// NewClient returns an initialized Client, this will also create any topics needed
func NewClient(project string, opts ...option.ClientOption) (*Client, error) {
	client, err := pubsub.NewClient(context.Background(), project, opts...)
	if err != nil {
		return nil, err
	}

	topic := client.Topic(RoomCommandsTopic)
	if exists, err := topic.Exists(context.Background()); err != nil {
		return nil, err
	} else if !exists {
		topic, err = client.CreateTopicWithConfig(context.Background(), RoomCommandsTopic, &pubsub.TopicConfig{})
		if err != nil {
			return nil, err
		}
	}

	c := &Client{p: client, topic: topic}

	return c, nil
}

// SendCommand will send any eh.Command to the command handler server
// Blocks until sent
func (c *Client) SendCommand(ctx context.Context, command eh.Command) error {
	data, err := json.Marshal(command)
	if err != nil {
		return err
	}

	fmt.Printf("Sending command: type: %v, content: %v\n", command.CommandType(), string(data))

	res := c.topic.Publish(ctx, &pubsub.Message{
		ID:   uuid.NewString(),
		Data: data,
		Attributes: map[string]string{
			CommandTypeAttributeKey: string(command.CommandType()),
		},
		PublishTime: time.Now(),
	})
	_, err = res.Get(ctx)
	if err != nil {
		return err
	}
	return nil
}
