package indexer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/hexsans/hexmagnet/internal/model"
	"github.com/hexsans/hexmagnet/internal/protocol"
)

var cleanNameRe = regexp.MustCompile(`\[[^\]]*\]|\([^)]*\d{4}[^)]*\)|\.\w{3,4}$`)

const IndexName = "torrent_content"

type TorrentContentDocument struct {
	ID            string    `json:"id"`
	InfoHash      string    `json:"info_hash"`
	Name          string    `json:"name"`
	Title         string    `json:"title"`
	Overview      string    `json:"overview"`
	ContentType   string    `json:"content_type"`
	ContentSource string    `json:"content_source"`
	ContentID     string    `json:"content_id"`
	Size          uint64    `json:"size"`
	Seeders       *uint     `json:"seeders"`
	Leechers      *uint     `json:"leechers"`
	FilesCount    *uint     `json:"files_count"`
	FileTypes     []string  `json:"file_types,omitempty"`
	Languages     []string  `json:"languages"`
	ReleaseYear   *uint16   `json:"release_year"`
	Popularity    *float32  `json:"popularity"`
	VoteAverage   *float32  `json:"vote_average"`
	VoteCount     *uint     `json:"vote_count"`
	Adult         *bool     `json:"adult"`
	Private       bool      `json:"private"`
	CreatedAt     string    `json:"created_at"`
	UpdatedAt     string    `json:"updated_at"`
	SearchVector  []float32 `json:"search_vector,omitempty"`

	// Files is used to build the search text and file type list. It is not
	// serialized (the mapping stores only the deduplicated FileTypes), which
	// avoids indexing one nested Lucene document per file.
	Files []File `json:"-"`
}

type File struct {
	PathParts []string `json:"path_parts"`
	Extension string   `json:"extension"`
	Size      uint64   `json:"size"`
}

type Attribute struct {
	Source string `json:"source"`
	Key    string `json:"key"`
	Value  string `json:"value"`
}

func NewDocument(t model.Torrent) TorrentContentDocument {
	doc := TorrentContentDocument{
		ID:        t.InfoHash.String(),
		InfoHash:  t.InfoHash.String(),
		Name:      t.Name,
		Size:      t.Size,
		Private:   t.Private,
		CreatedAt: t.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: t.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	if t.Seeders.Valid {
		doc.Seeders = &t.Seeders.Uint
	}

	if t.Leechers.Valid {
		doc.Leechers = &t.Leechers.Uint
	}

	if t.FilesCount.Valid {
		doc.FilesCount = &t.FilesCount.Uint
	}

	if t.ContentType.Valid {
		doc.ContentType = t.ContentType.ContentType.String()
	} else {
		doc.ContentType = "unknown"
	}

	if t.ContentSource.Valid {
		doc.ContentSource = t.ContentSource.String
	}

	if t.ContentID.Valid {
		doc.ContentID = t.ContentID.String
	}

	doc.Languages = make([]string, 0, len(t.Languages))
	for lang := range t.Languages {
		doc.Languages = append(doc.Languages, lang.String())
	}

	if t.ContentID.Valid {
		doc.Title = t.Content.Title

		if t.Content.Overview.Valid {
			doc.Overview = t.Content.Overview.String
		}

		if !t.Content.ReleaseDate.IsNil() {
			doc.ReleaseYear = new(uint16(t.Content.ReleaseDate.Year))
		}

		if t.Content.Adult.Valid {
			doc.Adult = &t.Content.Adult.Bool
		}

		if t.Content.Popularity.Valid {
			doc.Popularity = &t.Content.Popularity.Float32
		}

		if t.Content.VoteAverage.Valid {
			doc.VoteAverage = &t.Content.VoteAverage.Float32
		}

		if t.Content.VoteCount.Valid {
			doc.VoteCount = &t.Content.VoteCount.Uint
		}
	}

	doc.Files = make([]File, 0, len(t.Files))

	typeSeen := make(map[string]struct{})

	for _, f := range t.Files {
		doc.Files = append(doc.Files, File{
			PathParts: f.PathParts,
			Extension: f.Extension.String,
			Size:      f.Size,
		})

		if ext := strings.ToLower(strings.TrimSpace(f.Extension.String)); ext != "" {
			if ft := model.FileTypeFromExtension(ext); ft.Valid {
				key := ft.FileType.String()
				if _, ok := typeSeen[key]; !ok {
					typeSeen[key] = struct{}{}
					doc.FileTypes = append(doc.FileTypes, key)
				}
			}
		}
	}

	return doc
}

func IndexMapping(dims int) string {
	mapping := `{
  "settings": {
    "number_of_shards": 1,
    "number_of_replicas": 0,
    "index.codec": "best_compression"
  },
  "mappings": {
    "properties": {
      "id": { "type": "keyword" },
      "info_hash": { "type": "keyword" },
      "name": {
        "type": "text",
        "copy_to": ["search_text"]
      },
      "title": {
        "type": "text",
        "copy_to": ["search_text"]
      },
      "overview": {
        "type": "text",
        "copy_to": ["search_text"]
      },
      "search_text": { "type": "text" },
      "content_type": { "type": "keyword" },
      "content_source": { "type": "keyword" },
      "content_id": { "type": "keyword" },
      "size": { "type": "long" },
      "seeders": { "type": "integer" },
      "leechers": { "type": "integer" },
      "files_count": { "type": "integer" },
      "file_types": { "type": "keyword" },
      "languages": { "type": "keyword" },
      "release_year": { "type": "integer" },
      "popularity": { "type": "float" },
      "vote_average": { "type": "float" },
      "vote_count": { "type": "integer" },
      "adult": { "type": "boolean" },
      "private": { "type": "boolean" },
      "search_vector": {
        "type": "dense_vector",
        "dims": __DIMS__,
        "index": true,
        "similarity": "cosine",
        "index_options": {
          "type": "int8_hnsw"
        }
      },
      "created_at": { "type": "date" },
      "updated_at": { "type": "date" }
    }
  }
}`

	return strings.Replace(mapping, "__DIMS__", fmt.Sprintf("%d", dims), 1)
}

func parseCompositeID(id string) (infoHash protocol.ID, contentType string, contentSource string, contentID string, err error) {
	parts := strings.SplitN(id, ":", 4)
	if len(parts) != 4 {
		return protocol.ID{}, "", "", "", fmt.Errorf("invalid composite ID: %s", id)
	}

	infoHash, err = protocol.ParseID(parts[0])
	if err != nil {
		return protocol.ID{}, "", "", "", fmt.Errorf("invalid info hash in composite ID %s: %w", id, err)
	}

	return infoHash, parts[1], parts[2], parts[3], nil
}

// BuildSearchText builds the text used to generate the semantic embedding for a
// torrent. maxFiles bounds how many file names are included (<= 0 means no cap).
func BuildSearchText(doc TorrentContentDocument, maxFiles int) string {
	var parts []string

	// If we have content metadata, build a rich description
	if doc.Title != "" && doc.Title != doc.Name {
		contentParts := []string{doc.Title}
		if doc.ContentType != "" && doc.ContentType != "unknown" {
			contentParts = append(contentParts, fmt.Sprintf("(%s)", doc.ContentType))
		}

		if doc.ReleaseYear != nil {
			contentParts = append(contentParts, fmt.Sprintf("(%d)", *doc.ReleaseYear))
		}

		parts = append(parts, strings.Join(contentParts, " "))
		if doc.Overview != "" {
			parts = append(parts, doc.Overview)
		}
		// Also include the raw name for additional context
		cleaned := cleanNameRe.ReplaceAllString(doc.Name, "")

		cleaned = strings.TrimSpace(cleaned)
		if cleaned != "" && cleaned != doc.Title {
			parts = append(parts, cleaned)
		}
	} else {
		// No structured metadata — clean the raw name
		name := doc.Name
		if name == "" && doc.Title != "" {
			name = doc.Title
		}

		if name != "" {
			cleaned := cleanNameRe.ReplaceAllString(name, "")

			cleaned = strings.TrimSpace(cleaned)
			if cleaned != "" {
				parts = append(parts, cleaned)
			} else {
				parts = append(parts, name)
			}
		}

		if doc.Overview != "" {
			parts = append(parts, doc.Overview)
		}
	}

	if len(doc.Files) > 0 {
		sorted := make([]File, len(doc.Files))
		copy(sorted, doc.Files)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Size > sorted[j].Size
		})

		seen := map[string]bool{}
		fileCount := 0

		for _, f := range sorted {
			if len(f.PathParts) == 0 {
				continue
			}

			fileName := f.PathParts[len(f.PathParts)-1]
			cleaned := cleanNameRe.ReplaceAllString(fileName, "")

			cleaned = strings.TrimSpace(cleaned)
			if cleaned == "" || seen[cleaned] {
				continue
			}

			seen[cleaned] = true

			parts = append(parts, cleaned)
			fileCount++

			if maxFiles > 0 && fileCount >= maxFiles {
				break
			}
		}
	}

	return strings.Join(parts, " ")
}

func BulkBody(docs []TorrentContentDocument) ([]byte, error) {
	var buf bytes.Buffer

	for _, doc := range docs {
		meta := map[string]any{
			"index": map[string]any{
				"_index": IndexName,
				"_id":    doc.InfoHash,
			},
		}

		metaJSON, err := json.Marshal(meta)
		if err != nil {
			return nil, fmt.Errorf("marshal meta: %w", err)
		}

		_, _ = buf.Write(metaJSON)
		_ = buf.WriteByte('\n')

		docJSON, err := json.Marshal(doc)
		if err != nil {
			return nil, fmt.Errorf("marshal document: %w", err)
		}

		_, _ = buf.Write(docJSON)
		_ = buf.WriteByte('\n')
	}

	return buf.Bytes(), nil
}
