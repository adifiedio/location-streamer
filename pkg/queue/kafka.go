package queue

import (
	"context"
	"encoding/json"
	"log"

	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(brokers []string, topic string) *Producer {
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Topic:                  topic,
		Balancer:               &kafka.LeastBytes{},
		RequiredAcks:           kafka.RequireOne, // ensure it's actually written
		AllowAutoTopicCreation: true,
		Async:                  false, // block until written
		// add some logging for troubleshooting
		ErrorLogger: kafka.LoggerFunc(func(s string, i ...interface{}) {
			log.Printf("kafka producer error: "+s, i...)
		}),
	}
	return &Producer{writer: writer}
}

func (p *Producer) Produce(ctx context.Context, key string, value interface{}) error {
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(key),
		Value: jsonBytes,
	})
}

func (p *Producer) Close() error {
	return p.writer.Close()
}

type Consumer struct {
	reader *kafka.Reader
}

func NewConsumer(brokers []string, topic string, groupID string) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		Topic:       topic,
		GroupID:     groupID,
		MinBytes:    1,    // 1 byte - process as soon as we have any data
		MaxBytes:    10e6, // 10MB
		StartOffset: kafka.FirstOffset,
	})
	return &Consumer{reader: reader}
}

func (c *Consumer) Consume(ctx context.Context) (kafka.Message, error) {
	return c.reader.ReadMessage(ctx)
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}
