package torrentstore

import (
	"context"
	"fmt"

	"github.com/hexsans/hexmagnet/internal/configmgr"
	"go.uber.org/fx"
)

// NewModule wires the file-backed torrent store.
func NewModule() fx.Option {
	return fx.Module(
		"torrentstore",
		fx.Provide(
			func(cm *configmgr.Manager) *Store {
				return New(func() string {
					if cm == nil {
						return ""
					}

					return cm.Get().Server.TorrentFilePath
				})
			},
		),
		fx.Invoke(func(store *Store, cm *configmgr.Manager) error {
			if err := store.EnsureTorrentDir(); err != nil {
				return fmt.Errorf("ensure torrent file directory: %w", err)
			}

			if cm != nil {
				cm.Subscribe(context.Background(), "torrent_store",
					func(context.Context, *configmgr.Snapshot) error {
						return store.EnsureTorrentDir()
					}, configmgr.ApplyAsync)
			}

			return nil
		}),
	)
}
