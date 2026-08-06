package metainfo

type TorrentFile struct {
	Info Info `bencode:"info"`

	Announce     string     `bencode:"announce"`
	AnnounceList [][]string `bencode:"announce-list,omitempty"`
	CreationDate int64      `bencode:"creation date,omitempty"`
	Comment      string     `bencode:"comment,omitempty"`
	CreatedBy    string     `bencode:"created by,omitempty"`
	URLList      any        `bencode:"url-list,omitempty"`
}
