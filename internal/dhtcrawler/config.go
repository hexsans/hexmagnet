package dhtcrawler

import (
	"time"
)

type Config struct {
	BootstrapNodes               []string
	ReseedBootstrapNodesInterval time.Duration `validate:"gt=0s"`
	RescrapeThreshold            uint          `validate:"gte=1"`
	HashDiscoverLimit            int           `validate:"gte=1"`
	EmbedTrackers                []string
}

// https://github.com/anacrolix/dht/blob/92b36a3fa7a37a15e08684337b47d8d0fb322ab6/dht.go#L106
var defaultBootstrapNodes = []string{
	"router.utorrent.com:6881",
	"router.bittorrent.com:6881",
	"dht.transmissionbt.com:6881",
	"dht.aelitis.com:6881",     // Vuze
	"router.silotis.us:6881",   // IPv6
	"dht.libtorrent.org:25401", // @arvidn's
}
