package rabbitmq

import (
	"fmt"
	"time"

	"emotibot.com/emotigo/pkg/logger"
)

//Consumer is response for listening and taking message from the config queue.
type Consumer struct {
	client   *Client
	config   ConsumerConfig
	isClosed bool
}

type ConsumerConfig struct {
	QueueName string
	MaxRetry  int
}

// Task is the task that will be triggered if new message comes from queue.
type Task func(message []byte) error

// Subscribe will create a routine to check for the
func (c *Consumer) Subscribe(task Task) error {
	// Create queue if not exists
	ch, _ := c.client.rwChannels()
	_, err := ch.QueueDeclare(
		c.config.QueueName, // name
		false,              // durable
		false,              // delete when unused
		false,              // exclusive
		false,              // no-wait
		nil,                // arguments
	)
	if err != nil {
		e := fmt.Errorf("Fail to create queue: %s, error: %s",
			c.config.QueueName, err.Error())
		return e
	}

	go func() {
		for {
			ch, _ := c.client.rwChannels()
			msgs, err := ch.Consume(
				c.config.QueueName, // queue
				"",                 // consumer
				false,              // auto-ack
				false,              // exclusive
				false,              // no-local
				false,              // no-wait
				nil,                // args
			)

			if err != nil {
				logger.Warn.Println("failed to register a consumer, ", err)
				time.Sleep(time.Duration(3) * time.Second)
				c.client.reconnect()
				continue
			}
			for d := range msgs {
				err := task(d.Body)
				if err != nil {
					logger.Warn.Println("Failed to consume message: ", err)
					continue
				}
				d.Ack(false)
			}
			c.client.reconnect()
		}
	}()

	return nil
}

func (c *Consumer) Consume() ([]byte, error) {
	var err error
	maxRetry := c.config.MaxRetry
	for i := 0; ; i++ {
		if maxRetry > 0 && i == maxRetry {
			break
		}
		if i > 0 {
			time.Sleep(time.Duration(100) * time.Millisecond)
		}
		ch, _ := c.client.rwChannels()
		q, err := ch.QueueDeclare(
			c.config.QueueName, // name
			false,              // durable
			false,              // delete when unused
			false,              // exclusive
			false,              // no-wait
			nil,                // arguments
		)

		if err != nil {
			err = fmt.Errorf("failed to declare a queue, %v", err)
			continue
		}
		msg, ok, err := ch.Get(q.Name, true)
		if !ok {
			err = fmt.Errorf("queue is empty")
			continue
		}
		if err != nil {
			err = fmt.Errorf("get message from queue failed, %v", err)
			continue
		}
		return msg.Body, nil
	}
	return nil, fmt.Errorf("exceed max retries %d, error: %v", c.config.MaxRetry, err)
}
