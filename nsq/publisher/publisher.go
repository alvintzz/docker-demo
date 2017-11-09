package publisher

import (
	"encoding/json"
	"fmt"
	"time"
	"context"

	"github.com/bitly/go-nsq"
)

type ProducerQueue interface {
	Publish(ctx context.Context, topic string, data interface{}) error
}

type ProducerNSQ struct {
	publisher *nsq.Producer
}

func AddPublisher(address string) (ProducerNSQ, error) {
	config := nsq.NewConfig()
	config.DefaultRequeueDelay = 0
	config.MaxBackoffDuration = time.Millisecond * 50
	config.MaxInFlight = 50
	config.MaxAttempts = 10

	producer, err := nsq.NewProducer(address, config)
	if err != nil {
		err = fmt.Errorf("Failed to create new producer because %s", err.Error())
	}

	pub := ProducerNSQ{
		publisher: producer,
	}

	return pub, err
}

func (prod ProducerNSQ) Publish(ctx context.Context, topic string, data interface{}) error {
	message, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("Failed to Marshal NSQ Data because %s", err.Error())
	}

	err = prod.publisher.Publish(topic, message)
	if err != nil {
		return fmt.Errorf("Failed to publish nsq message because %s", err.Error())
	}

	return nil
}