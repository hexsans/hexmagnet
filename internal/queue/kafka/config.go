package kafka

type Config struct {
	Brokers []string `validate:"min=1"`
}

func DefaultConfig() Config {
	return Config{
		Brokers: []string{"localhost:9092"},
	}
}
