package db

import sqlc "github.com/hexsans/hexmagnet/internal/database/db/sqlc"

type (
	BlockedInfoHash = sqlc.BlockedInfoHash
	Content         = sqlc.Content
	KeyValue        = sqlc.KeyValue
	Torrent         = sqlc.Torrent
	TorrentFile     = sqlc.TorrentFile
)

type (
	GetContentParams                  = sqlc.GetContentParams
	ListTorrentFilesPaginatedParams   = sqlc.ListTorrentFilesPaginatedParams
	ListTorrentsPaginatedParams       = sqlc.ListTorrentsPaginatedParams
	ListTorrentsPaginatedBeforeParams = sqlc.ListTorrentsPaginatedBeforeParams
	TorrentFileExistsParams           = sqlc.TorrentFileExistsParams
	UpsertBlockedInfoHashParams       = sqlc.UpsertBlockedInfoHashParams
	UpsertContentParams               = sqlc.UpsertContentParams
	UpsertKeyValueParams              = sqlc.UpsertKeyValueParams
	UpsertTorrentFileParams           = sqlc.UpsertTorrentFileParams
	UpsertTorrentParams               = sqlc.UpsertTorrentParams
	UpdateTorrentSeedersParams        = sqlc.UpdateTorrentSeedersParams
	UpdateTorrentContentParams        = sqlc.UpdateTorrentContentParams
)
