package consumer

import (
	"github.com/bitly/go-nsq"
)

type subscriber struct {
	topic      string
	channel    string
	handler    nsq.HandlerFunc
}

type ConsumerManager struct {
	nsqLocation string
	consumer    []subscriber
}

func NewConsumerManager(address string) *ConsumerManager {
	return &ConsumerManager{
		nsqLocation: address,
	}
}

func (cm *ConsumerManager) Register(topic, channel string, handler nsq.HandlerFunc) {
	sub := subscriber{
		topic: topic,
		channel: channel,
		handler: handler,
	}
	cm.consumer = append(cm.consumer, sub)
}

func (cm *ConsumerManager) Run() error {
	config := nsq.NewConfig()

	for _, args := range cm.consumer {
		consumer, _ := nsq.NewConsumer(args.topic, args.channel, config)
		consumer.AddHandler(args.handler)

		err := consumer.ConnectToNSQD(cm.nsqLocation)
		if err != nil {
			return err
		}
	}

	return nil
}