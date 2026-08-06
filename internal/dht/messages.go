package dht

import "time"

type DiscoveredHash struct {
	InfoHash     string    `json:"info_hash"`
	Node         string    `json:"node"`
	DiscoveredAt time.Time `json:"discovered_at"`
}

type GetPeersMessage struct {
	InfoHash string `json:"info_hash"`
	Node     string `json:"node"`
}

type ScrapeMessage struct {
	InfoHash string `json:"info_hash"`
	Node     string `json:"node"`
}

type MetainfoFile struct {
	PathParts []string `json:"path_parts"`
	Size      uint64   `json:"size"`
}

type MetaInfoMessage struct {
	InfoHash     string         `json:"info_hash"`
	RawInfoBytes []byte         `json:"raw_info_bytes"`
	Name         string         `json:"name"`
	Private      bool           `json:"private"`
	Files        []MetainfoFile `json:"files"`
	TotalSize    uint64         `json:"total_size"`
	Node         string         `json:"node"`
}

type ScrapeResultMessage struct {
	InfoHash  string    `json:"info_hash"`
	Seeders   uint      `json:"seeders"`
	Leechers  uint      `json:"leechers"`
	ScrapedAt time.Time `json:"scraped_at"`
}
