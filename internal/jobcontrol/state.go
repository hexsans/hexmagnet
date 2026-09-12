package jobcontrol

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/hexsans/hexmagnet/internal/database/db"
	"github.com/hexsans/hexmagnet/internal/utils"
	"github.com/jackc/pgx/v5"
)

const (
	KeyReindexState    = "maintenance.reindex.state"
	KeyReclassifyState = "maintenance.reclassify.state"
)

// State is the persisted progress of a resumable maintenance job. It is stored
// as JSON in the key_value table so reindex and reclassify can continue from
// the last processed torrent after a restart.
type State struct {
	BarrierTime     time.Time `json:"barrier_time"`
	CursorCreatedAt time.Time `json:"cursor_created_at"`
	CursorInfoHash  string    `json:"cursor_info_hash"`
	Total           int       `json:"total"`
	Count           int       `json:"count"`
	Done            bool      `json:"done"`
	Fingerprint     string    `json:"fingerprint"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// StateStore persists and restores maintenance job progress. Load reports
// whether a state was found.
type StateStore interface {
	Load(ctx context.Context, key string) (State, bool, error)
	Save(ctx context.Context, key string, state State) error
	Delete(ctx context.Context, key string) error
}

type keyValueStateStore struct {
	queries utils.Lazy[*db.Queries]
}

func NewStateStore(queries utils.Lazy[*db.Queries]) StateStore {
	return &keyValueStateStore{queries: queries}
}

func (s *keyValueStateStore) Load(ctx context.Context, key string) (State, bool, error) {
	q, err := s.queries.Get()
	if err != nil {
		return State{}, false, err
	}

	record, err := q.GetKeyValue(ctx, key)
	if errors.Is(err, pgx.ErrNoRows) {
		return State{}, false, nil
	}

	if err != nil {
		return State{}, false, err
	}

	var state State
	if err := json.Unmarshal(record.Value, &state); err != nil {
		return State{}, false, err
	}

	return state, true, nil
}

func (s *keyValueStateStore) Save(ctx context.Context, key string, state State) error {
	q, err := s.queries.Get()
	if err != nil {
		return err
	}

	state.UpdatedAt = time.Now().UTC()

	data, err := json.Marshal(state)
	if err != nil {
		return err
	}

	return q.UpsertKeyValue(ctx, db.UpsertKeyValueParams{Key: key, Value: data})
}

func (s *keyValueStateStore) Delete(ctx context.Context, key string) error {
	q, err := s.queries.Get()
	if err != nil {
		return err
	}

	return q.DeleteKeyValue(ctx, key)
}

// Fingerprint hashes the given value so configuration changes can be detected
// across restarts. It returns an empty string when the value cannot be hashed.
func Fingerprint(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}

	sum := sha256.Sum256(data)

	return hex.EncodeToString(sum[:])
}
