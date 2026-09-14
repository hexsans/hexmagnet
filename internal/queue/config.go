package queue

import (
	"time"

	"github.com/hexsans/hexmagnet/internal/queue/delivery"
	kafka2 "github.com/hexsans/hexmagnet/internal/queue/kafka"
)

type Config struct {
	Backend string        `yaml:"backend"`
	Kafka   kafka2.Config `yaml:"kafka"`

	// MaxDeliveryAttempts is the number of times a message is delivered to a
	// handler before it is treated as poison and discarded. Permanent errors
	// are discarded immediately regardless of this limit.
	MaxDeliveryAttempts int `yaml:"max_delivery_attempts"`

	// DeliveryBackoff is the base delay before the first redelivery; it grows
	// exponentially up to a 30s cap.
	DeliveryBackoff time.Duration `yaml:"delivery_backoff"`
}

const (
	backendMemory = "memory"
	backendKafka  = "kafka"
)

func defaultBackend() string {
	return backendMemory
}

func NewDefaultConfig() Config {
	return Config{
		Backend:             defaultBackend(),
		Kafka:               kafka2.DefaultConfig(),
		MaxDeliveryAttempts: delivery.DefaultMaxAttempts,
		DeliveryBackoff:     delivery.DefaultBackoffBase,
	}
}

// deliveryConfig converts the queue-level settings into the shared delivery
// policy used by the backends.
func (c Config) deliveryConfig() delivery.Config {
	cfg := delivery.DefaultConfig()

	if c.MaxDeliveryAttempts > 0 {
		cfg.MaxAttempts = c.MaxDeliveryAttempts
	}

	if c.DeliveryBackoff > 0 {
		cfg.Backoff.Base = c.DeliveryBackoff
	}

	return cfg
}
