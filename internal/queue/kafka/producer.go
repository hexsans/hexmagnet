package kafka

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/IBM/sarama"
	"github.com/hexsans/hexmagnet/internal/worker"
	"go.uber.org/zap"
)

type Producer struct {
	producer sarama.AsyncProducer
	logger   *zap.SugaredLogger
}

func NewProducer(brokers []string, logger *zap.SugaredLogger) (*Producer, error) {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForLocal
	config.Producer.Return.Successes = false
	config.Producer.Return.Errors = true
	config.Producer.Flush.Frequency = 500 * time.Millisecond
	config.Producer.Partitioner = sarama.NewHashPartitioner
	config.Producer.MaxMessageBytes = 10 * 1024 * 1024

	producer, err := sarama.NewAsyncProducer(brokers, config)
	if err != nil {
		return nil, err
	}

	p := &Producer{
		producer: producer,
		logger:   logger,
	}

	worker.GoRecover(p.logger, "kafka_producer_errors", p.handleErrors)

	return p, nil
}

func (p *Producer) handleErrors() {
	for err := range p.producer.Errors() {
		p.logger.Errorw("kafka producer error",
			"topic", err.Msg.Topic,
			"key", fmt.Sprintf("%v", err.Msg.Key),
			"error", err,
		)
	}
}

func (p *Producer) Produce(topic string, key string, value any) {
	data, err := json.Marshal(value)
	if err != nil {
		p.logger.Errorw("failed to marshal kafka message", "error", err)
		return
	}

	p.producer.Input() <- &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.StringEncoder(key),
		Value: sarama.ByteEncoder(data),
	}
}

func (p *Producer) Close() error {
	return p.producer.Close()
}
