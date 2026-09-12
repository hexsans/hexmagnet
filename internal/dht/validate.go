package dht

import (
	"github.com/hexsans/hexmagnet/internal/protocol"
	"github.com/hexsans/hexmagnet/internal/queue/permanent"
	"go.uber.org/zap"
)

func ParseInfoHash(logger *zap.SugaredLogger, infoHash string, component string) (protocol.ID, error) {
	id, err := protocol.ParseID(infoHash)
	if err != nil {
		logger.Debugw("failed to parse info hash in "+component, "info_hash", infoHash, "error", err)
		return protocol.ID{}, permanent.Mark(err)
	}

	return id, nil
}
