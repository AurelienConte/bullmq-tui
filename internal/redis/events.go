package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type EventListener struct {
	client    redis.Cmdable
	prefix    string
	queues    []string
	eventChan chan JobEvent
	stopChan  chan struct{}
}

func NewEventListener(client redis.Cmdable, prefix string) *EventListener {
	return &EventListener{
		client:    client,
		prefix:    prefix,
		eventChan: make(chan JobEvent, 100),
		stopChan:  make(chan struct{}),
	}
}

func (e *EventListener) Subscribe(ctx context.Context, queues []string) (<-chan JobEvent, error) {
	e.queues = queues

	// Build channel patterns: bull:<queue>:events for each queue
	patterns := make([]string, len(queues))
	for i, q := range queues {
		patterns[i] = fmt.Sprintf("%s:%s:events", e.prefix, q)
	}

	// Get PubSub interface
	var pubsub *redis.PubSub

	// Check if client supports PubSub
	switch c := e.client.(type) {
	case *redis.Client:
		pubsub = c.PSubscribe(ctx, patterns...)
	case *redis.ClusterClient:
		pubsub = c.PSubscribe(ctx, patterns...)
	default:
		return nil, fmt.Errorf("client does not support pub/sub")
	}

	go func() {
		defer pubsub.Close()

		ch := pubsub.Channel()
		for {
			select {
			case msg := <-ch:
				if msg == nil {
					continue
				}
				event := e.parseEvent(msg)
				if event != nil {
					e.eventChan <- *event
				}
			case <-e.stopChan:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	return e.eventChan, nil
}

func (e *EventListener) parseEvent(msg *redis.Message) *JobEvent {
	// Parse BullMQ event format
	// Events are JSON: {"event": "completed", "jobId": "123", ...}
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(msg.Payload), &data); err != nil {
		return nil
	}

	// Extract queue name from channel: bull:<queue>:events
	parts := strings.Split(msg.Channel, ":")
	if len(parts) < 3 {
		return nil
	}
	queueName := parts[1]

	event := &JobEvent{
		QueueName: queueName,
		Timestamp: time.Now(),
		Data:      data,
	}

	if eventType, ok := data["event"].(string); ok {
		event.Event = eventType
	}

	if jobID, ok := data["jobId"].(string); ok {
		event.JobID = jobID
	}

	return event
}

func (e *EventListener) Stop() {
	close(e.stopChan)
	close(e.eventChan)
}
