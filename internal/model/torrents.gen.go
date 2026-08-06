package model

import (
	"time"

	"github.com/hexsans/hexmagnet/internal/database/fts"
	"github.com/hexsans/hexmagnet/internal/protocol"
)

const TableNameTorrent = "torrents"

type Torrent struct {
	InfoHash      protocol.ID     `gorm:"column:info_hash;primaryKey;<-:create"                                    json:"infoHash"`
	Name          string          `gorm:"column:name;not null"                                                     json:"name"`
	Size          uint64          `gorm:"column:size;not null"                                                     json:"size"`
	Private       bool            `gorm:"column:private;not null"                                                  json:"private"`
	CreatedAt     time.Time       `gorm:"column:created_at;not null;<-:create"                                     json:"createdAt"`
	UpdatedAt     time.Time       `gorm:"column:updated_at;not null"                                               json:"updatedAt"`
	FilesCount    NullUint        `gorm:"column:files_count"                                                       json:"filesCount"`
	ContentType   NullContentType `gorm:"column:content_type"                                                      json:"contentType"`
	ContentSource NullString      `gorm:"column:content_source"                                                    json:"contentSource"`
	ContentID     NullString      `gorm:"column:content_id"                                                        json:"contentId"`
	Languages     Languages       `gorm:"column:languages;serializer:json"                                         json:"languages"`
	Tsv           fts.Tsvector    `gorm:"column:tsv"                                                               json:"tsv"`
	Seeders       NullUint        `gorm:"column:seeders"                                                           json:"seeders"`
	Leechers      NullUint        `gorm:"column:leechers"                                                          json:"leechers"`
	Files         []TorrentFile   `gorm:"foreignKey:InfoHash"                                                      json:"files"`
	Content       Content         `gorm:"foreignKey:ContentType,ContentSource,ContentID;references:Type,Source,ID" json:"content"`
}

func (*Torrent) TableName() string {
	return TableNameTorrent
}
