package messaging

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// Producer는 Kafka로 메시지를 전송하는 헬퍼입니다.
type Producer struct {
	writer *kafka.Writer
	logger *zap.SugaredLogger
	topic  string
}

// NewProducer는 새로운 Kafka Producer를 생성합니다.
func NewProducer(brokers []string, topic string, logger *zap.SugaredLogger) (*Producer, error) {
	if len(brokers) == 0 || brokers[0] == "" {
		return nil, errors.New("brokers must not be empty")
	}
	if topic == "" {
		return nil, errors.New("topic must not be empty")
	}
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireAll,
		Async:        false,
		BatchTimeout: 500 * time.Millisecond,
	}

	return &Producer{
		writer: writer,
		logger: logger,
		topic:  topic,
	}, nil
}

// Publish는 단일 메시지를 Kafka에 전달합니다.
func (p *Producer) Publish(ctx context.Context, key []byte, value []byte) error {
	if p == nil || p.writer == nil {
		return errors.New("producer is not initialized")
	}
	msg := kafka.Message{
		Key:   key,
		Value: value,
		Time:  time.Now().UTC(),
	}
	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("write kafka message: %w", err)
	}
	p.logger.Debugw("published kafka message", "topic", p.topic, "key", string(key))
	return nil
}

// Close는 writer 자원을 해제합니다.
func (p *Producer) Close() error {
	if p == nil || p.writer == nil {
		return nil
	}
	return p.writer.Close()
}

// ConsumerConfig는 Kafka Consumer 설정입니다.
type ConsumerConfig struct {
	Brokers []string
	Topic   string
	GroupID string
}

// Consumer는 Kafka 메시지를 pull 방식으로 가져오는 헬퍼입니다.
type Consumer struct {
	reader *kafka.Reader
	logger *zap.SugaredLogger
}

// NewConsumer는 새로운 Kafka Consumer를 생성합니다.
func NewConsumer(cfg ConsumerConfig, logger *zap.SugaredLogger) (*Consumer, error) {
	if len(cfg.Brokers) == 0 || cfg.Brokers[0] == "" {
		return nil, errors.New("brokers must not be empty")
	}
	if cfg.Topic == "" {
		return nil, errors.New("topic must not be empty")
	}
	if cfg.GroupID == "" {
		return nil, errors.New("group id must not be empty")
	}
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  cfg.Brokers,
		Topic:    cfg.Topic,
		GroupID:  cfg.GroupID,
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})

	return &Consumer{
		reader: reader,
		logger: logger,
	}, nil
}

// Fetch는 다음 메시지를 가져옵니다.
func (c *Consumer) Fetch(ctx context.Context) (kafka.Message, error) {
	if c == nil || c.reader == nil {
		return kafka.Message{}, errors.New("consumer is not initialized")
	}
	msg, err := c.reader.FetchMessage(ctx)
	if err != nil {
		return kafka.Message{}, fmt.Errorf("fetch kafka message: %w", err)
	}
	return msg, nil
}

// Commit은 처리 완료된 메시지를 커밋합니다.
func (c *Consumer) Commit(ctx context.Context, msg kafka.Message) error {
	if c == nil || c.reader == nil {
		return errors.New("consumer is not initialized")
	}
	if err := c.reader.CommitMessages(ctx, msg); err != nil {
		return fmt.Errorf("commit kafka message: %w", err)
	}
	return nil
}

// Close는 reader 자원을 해제합니다.
func (c *Consumer) Close() error {
	if c == nil || c.reader == nil {
		return nil
	}
	return c.reader.Close()
}
