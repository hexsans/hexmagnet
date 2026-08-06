package dht

import "time"

type ResponderConfig struct {
	Enabled         bool `yaml:"enabled"`
	GlobalRateLimit int  `validate:"gte=0" yaml:"global_rate_limit"`
	PerIPRateLimit  int  `validate:"gte=0" yaml:"per_ip_rate_limit"`
}

type Config struct {
	Port                         uint16          `validate:"gte=1,lte=65535" yaml:"port"`
	Responder                    ResponderConfig `                           yaml:"responder"`
	BootstrapNodes               []string        `                           yaml:"bootstrap_nodes"`
	ReseedBootstrapNodesInterval time.Duration   `validate:"gt=0s"           yaml:"reseed_bootstrap_nodes_interval"`
}

func NewDefaultConfig() Config {
	return Config{
		Port: 3334,
		Responder: ResponderConfig{
			Enabled:         true,
			GlobalRateLimit: 50,
			PerIPRateLimit:  1,
		},
		BootstrapNodes:               defaultBootstrapNodes,
		ReseedBootstrapNodesInterval: time.Minute,
	}
}

var defaultBootstrapNodes = []string{
	"router.utorrent.com:6881",
	"router.bittorrent.com:6881",
	"dht.transmissionbt.com:6881",
	"dht.aelitis.com:6881",
	"router.silotis.us:6881",
	"dht.libtorrent.org:25401",
}
