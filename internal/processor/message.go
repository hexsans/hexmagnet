package processor

import (
	"github.com/hexsans/hexmagnet/internal/protocol"
)

type ClassifyMode int

const (
	ClassifyModeDefault ClassifyMode = iota
	ClassifyModeRematch
)

type MessageParams struct {
	ClassifyMode ClassifyMode  `json:"classify_mode,omitempty"`
	ContentType  string        `json:"content_type,omitempty"`
	InfoHashes   []protocol.ID `json:"info_hashes"`
}
