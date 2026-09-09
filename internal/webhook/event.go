package webhook

import (
	"encoding/json"
	"slices"
	"strings"
	"time"

	"github.com/hexsans/hexmagnet/internal/model"
	"github.com/hexsans/hexmagnet/internal/utils"
)

// Event is the payload delivered to every configured webhook URL.
type Event struct {
	// Event is the event type (see EventClassified).
	Event string `json:"event"`
	// InfoHash is the torrent's info hash.
	InfoHash string `json:"info_hash"`
	// Name is the torrent name as crawled.
	Name string `json:"name"`
	// Size is the total size in bytes.
	Size int64 `json:"size"`
	// ContentType is the classified content type (movie, tv_show, music, ...).
	ContentType *string `json:"content_type,omitempty"`
	// ContentSource / ContentID identify the external content when known
	// (tmdb, imdb, tvdb).
	ContentSource *string `json:"content_source,omitempty"`
	ContentID     *string `json:"content_id,omitempty"`
	// Languages are the detected languages of the torrent.
	Languages  []string `json:"languages,omitempty"`
	Seeders    *int32   `json:"seeders,omitempty"`
	Leechers   *int32   `json:"leechers,omitempty"`
	FilesCount *int32   `json:"files_count,omitempty"`
	// Files are the torrent's file paths (parts joined with "/").
	Files []string `json:"files,omitempty"`
	// Magnet is the magnet URI for the torrent.
	Magnet string `json:"magnet"`
	// TorrentURL is the .torrent download link. Only set when webhooks.base_url
	// is configured.
	TorrentURL string `json:"torrent_url,omitempty"`
	// CreatedAt is when the torrent was first seen.
	CreatedAt time.Time `json:"created_at"`
}

// NewClassifiedEvent builds the event emitted after a torrent is classified
// and persisted.
func NewClassifiedEvent(t model.Torrent) Event {
	e := Event{
		Event:     EventClassified,
		InfoHash:  t.InfoHash.String(),
		Name:      t.Name,
		Size:      int64(t.Size),
		Magnet:    t.MagnetURI(),
		CreatedAt: t.CreatedAt,
	}

	if t.ContentType.Valid {
		ct := t.ContentType.ContentType.String()
		e.ContentType = &ct
	}

	if t.ContentSource.Valid {
		src := t.ContentSource.String
		e.ContentSource = &src
	}

	if t.ContentID.Valid {
		id := t.ContentID.String
		e.ContentID = &id
	}

	if t.Seeders.Valid {
		s := utils.ClampInt32(t.Seeders.Uint)
		e.Seeders = &s
	}

	if t.Leechers.Valid {
		l := utils.ClampInt32(t.Leechers.Uint)
		e.Leechers = &l
	}

	if t.FilesCount.Valid {
		f := utils.ClampInt32(t.FilesCount.Uint)
		e.FilesCount = &f
	}

	if len(t.Languages) > 0 {
		e.Languages = make([]string, 0, len(t.Languages))
		for lang := range t.Languages {
			e.Languages = append(e.Languages, string(lang))
		}

		slices.Sort(e.Languages)
	}

	if len(t.Files) > 0 {
		e.Files = make([]string, 0, len(t.Files))
		for _, f := range t.Files {
			e.Files = append(e.Files, strings.Join(f.PathParts, "/"))
		}
	}

	return e
}

// JSON serializes the event.
func (e Event) JSON() ([]byte, error) {
	return json.Marshal(e)
}
