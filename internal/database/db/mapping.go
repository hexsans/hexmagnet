package db

import (
	"encoding/json"
	"fmt"

	"github.com/hexsans/hexmagnet/internal/database/fts"
	"github.com/hexsans/hexmagnet/internal/model"
	"github.com/hexsans/hexmagnet/internal/protocol"
	"github.com/jackc/pgx/v5/pgtype"
)

func ToProtocolID(s string) protocol.ID {
	id, err := protocol.ParseID(s)
	if err != nil {
		panic(fmt.Sprintf("ToProtocolID: invalid hex hash %q: %v", s, err))
	}

	return id
}

func FromProtocolID(id protocol.ID) string {
	return id.String()
}

func FromUintToInt32Ptr(v uint) *int32 {
	n := int32(v)
	return &n
}

func toNullUint(v *int32) model.NullUint {
	if v == nil {
		return model.NullUint{}
	}

	return model.NewNullUint(uint(*v))
}

func fromNullUint(v model.NullUint) *int32 {
	if !v.Valid {
		return nil
	}

	i := int32(v.Uint)

	return &i
}

func fromNullUint64(nu model.NullUint) *int64 {
	if !nu.Valid {
		return nil
	}

	v := int64(nu.Uint)

	return &v
}

func fromNullFloat32(nf model.NullFloat32) *float64 {
	if !nf.Valid {
		return nil
	}

	v := float64(nf.Float32)

	return &v
}

func TorrentFileToModel(tf TorrentFile) model.TorrentFile {
	return model.TorrentFile{
		InfoHash:  ToProtocolID(tf.InfoHash),
		Index:     uint(tf.Index),
		PathParts: tf.PathParts,
		Extension: model.NewNullStringFromPtr(tf.Extension),
		Size:      uint64(tf.Size),
		CreatedAt: tf.CreatedAt.Time,
		UpdatedAt: tf.UpdatedAt.Time,
	}
}

func TorrentToModel(t Torrent) model.Torrent {
	return model.Torrent{
		InfoHash:      ToProtocolID(t.InfoHash),
		Name:          t.Name,
		Size:          uint64(t.Size),
		Private:       t.Private,
		FilesCount:    toNullUint(t.FilesCount),
		ContentType:   contentTypeFromPtr(t.ContentType),
		ContentSource: model.NewNullStringFromPtr(t.ContentSource),
		ContentID:     model.NewNullStringFromPtr(t.ContentID),
		Languages:     languagesFromBytes(t.Languages),
		Tsv:           parseTsv(t.Tsv),
		Seeders:       toNullUint(t.Seeders),
		Leechers:      toNullUint(t.Leechers),
		CreatedAt:     t.CreatedAt.Time,
		UpdatedAt:     t.UpdatedAt.Time,
	}
}

func TorrentToUpsertParams(t model.Torrent) UpsertTorrentParams {
	return UpsertTorrentParams{
		InfoHash:   FromProtocolID(t.InfoHash),
		Name:       t.Name,
		Size:       int64(t.Size),
		Private:    t.Private,
		FilesCount: fromNullUint(t.FilesCount),
	}
}

func TorrentToUpdateContentParams(t model.Torrent) UpdateTorrentContentParams {
	return UpdateTorrentContentParams{
		InfoHash:      FromProtocolID(t.InfoHash),
		ContentType:   nullContentTypePtr(t.ContentType),
		ContentSource: t.ContentSource.Ptr(),
		ContentID:     t.ContentID.Ptr(),
		Languages:     languagesJSON(t.Languages),
		Tsv:           t.Tsv.String(),
	}
}

func ContentToUpsertParams(c model.Content) UpsertContentParams {
	return UpsertContentParams{
		Type:        string(c.Type),
		Source:      c.Source,
		ID:          c.ID,
		Title:       c.Title,
		ReleaseDate: dateToPGDate(c.ReleaseDate),
		Adult:       c.Adult.Ptr(),
		Overview:    c.Overview.Ptr(),
		Popularity:  fromNullFloat32(c.Popularity),
		VoteAverage: fromNullFloat32(c.VoteAverage),
		VoteCount:   fromNullUint64(c.VoteCount),
		Tsv:         c.Tsv.String(),
	}
}

func nullContentTypePtr(v model.NullContentType) *string {
	if !v.Valid {
		return nil
	}

	s := string(v.ContentType)

	return &s
}

func contentTypeFromPtr(s *string) model.NullContentType {
	if s == nil {
		return model.NullContentType{}
	}

	ct, err := model.ParseContentType(*s)
	if err != nil {
		return model.NullContentType{}
	}

	return model.NewNullContentType(ct)
}

func languagesJSON(v model.Languages) []byte {
	if v == nil {
		return nil
	}

	b, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("languagesJSON: %v", err))
	}

	return b
}

func languagesFromBytes(b []byte) model.Languages {
	if len(b) == 0 {
		return nil
	}

	var langs model.Languages
	if err := langs.UnmarshalJSON(b); err != nil {
		return nil
	}

	return langs
}

func parseTsv(s string) fts.Tsvector {
	t, _ := fts.ParseTsvector(s)
	return t
}

func dateToPGDate(d model.Date) pgtype.Date {
	if d.IsNil() {
		return pgtype.Date{Valid: false}
	}

	return pgtype.Date{Time: d.Time(), Valid: true}
}

func TorrentFileToUpsertParams(tf model.TorrentFile) UpsertTorrentFileParams {
	return UpsertTorrentFileParams{
		InfoHash:  FromProtocolID(tf.InfoHash),
		Index:     int32(tf.Index),
		PathParts: tf.PathParts,
		Size:      int64(tf.Size),
	}
}
