package metainforequester

type Config struct {
	RequestLimit      int  `validate:"gte=0" yaml:"request_limit"`
	RescrapeThreshold uint `validate:"gte=1" yaml:"rescrape_threshold"`
	HashDiscoverLimit int  `validate:"gte=1" yaml:"hash_discover_limit"`
}

func NewDefaultConfig() Config {
	return Config{
		RequestLimit:      50,
		RescrapeThreshold: 2592000,
		HashDiscoverLimit: 10,
	}
}
