package db

import sqlc "github.com/hexsans/hexmagnet/internal/database/db/sqlc"

type (
	BlockedInfoHash   = sqlc.BlockedInfoHash
	Content           = sqlc.Content
	KeyValue          = sqlc.KeyValue
	Torrent           = sqlc.Torrent
	TorrentFile       = sqlc.TorrentFile
	TorrentRetryQueue = sqlc.TorrentRetryQueue
)

type (
	GetContentParams                  = sqlc.GetContentParams
	BackoffTorrentRetryBatchParams    = sqlc.BackoffTorrentRetryBatchParams
	ListTorrentFilesPaginatedParams   = sqlc.ListTorrentFilesPaginatedParams
	ListTorrentsPaginatedParams       = sqlc.ListTorrentsPaginatedParams
	ListTorrentsPaginatedBeforeParams = sqlc.ListTorrentsPaginatedBeforeParams
	ListTorrentRetryQueueParams       = sqlc.ListTorrentRetryQueueParams
	MarkTorrentRetryDispatchedParams  = sqlc.MarkTorrentRetryDispatchedParams
	TorrentFileExistsParams           = sqlc.TorrentFileExistsParams
	UpsertBlockedInfoHashParams       = sqlc.UpsertBlockedInfoHashParams
	UpsertContentParams               = sqlc.UpsertContentParams
	UpsertKeyValueParams              = sqlc.UpsertKeyValueParams
	UpsertTorrentFileParams           = sqlc.UpsertTorrentFileParams
	UpsertTorrentParams               = sqlc.UpsertTorrentParams
	UpsertTorrentRetryBatchParams     = sqlc.UpsertTorrentRetryBatchParams
	UpsertTorrentRetryBatchRow        = sqlc.UpsertTorrentRetryBatchRow
	UpdateTorrentSeedersParams        = sqlc.UpdateTorrentSeedersParams
	UpdateTorrentContentParams        = sqlc.UpdateTorrentContentParams
)
