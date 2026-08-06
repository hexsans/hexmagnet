package dht

import (
	"github.com/hexsans/hexmagnet/internal/protocol"
	"go.uber.org/zap"
)

func ParseInfoHash(logger *zap.SugaredLogger, infoHash string, component string) (protocol.ID, error) {
	id, err := protocol.ParseID(infoHash)
	if err != nil {
		logger.Warnw("failed to parse info hash in "+component, "info_hash", infoHash, "error", err)
		return protocol.ID{}, err
	}

	return id, nil
}
