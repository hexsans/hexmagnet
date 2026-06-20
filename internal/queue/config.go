package queue

import kafka2 "github.com/hexsans/hexmagnet/internal/queue/kafka"

type Config struct {
	Backend string        `yaml:"backend"`
	Kafka   kafka2.Config `yaml:"kafka"`
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
		Backend: defaultBackend(),
		Kafka:   kafka2.DefaultConfig(),
	}
}
