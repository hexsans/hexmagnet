package elasticsearch

import "github.com/hexsans/hexmagnet/internal/elasticsearch/embedding"

const defaultAddress = "http://localhost:9200"

type Config struct {
	Addresses []string         `validate:"min=1" yaml:"addresses"`
	Embedding embedding.Config `                 yaml:"embedding"`
}

func DefaultConfig() Config {
	return Config{
		Addresses: []string{defaultAddress},
		Embedding: embedding.DefaultConfig(),
	}
}
