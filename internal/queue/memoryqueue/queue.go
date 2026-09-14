package memoryqueue

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hexsans/hexmagnet/internal/database/db"
	"github.com/hexsans/hexmagnet/internal/queue/delivery"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

const (
	memqueueStateKey = "memqueue.state"
	defaultCapacity  = 10_000
)

type Message struct {
	Seq   uint64          `json:"seq"`
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
}

type topicState struct {
	Messages  []*Message
	NextSeq   uint64
	Capacity  int
	Head      int
	Tail      int
	Consumers []*consumerState
}

type consumerState struct {
	GroupID string
	Offset  uint64
	Ch      chan *Message
	Cancel  context.CancelFunc
}

type MemoryQueue struct {
	mu        sync.RWMutex
	topics    map[string]*topicState
	offsets   map[string]map[string]uint64
	logger    *zap.SugaredLogger
	pool      *pgxpool.Pool
	persistCh chan struct{}
	stopCh    chan struct{}
	stopped   bool
	dirty     bool
	delivery  atomic.Pointer[delivery.Config]
}

// SetDelivery overrides the delivery policy used by consumers of this queue.
func (mq *MemoryQueue) SetDelivery(cfg delivery.Config) {
	normalized := cfg.Normalize()
	mq.delivery.Store(&normalized)
}

func (mq *MemoryQueue) deliveryConfig() delivery.Config {
	if p := mq.delivery.Load(); p != nil {
		return *p
	}

	return delivery.DefaultConfig()
}

func New(logger *zap.SugaredLogger) *MemoryQueue {
	return &MemoryQueue{
		topics:    make(map[string]*topicState),
		offsets:   make(map[string]map[string]uint64),
		logger:    logger.Named("memory_queue"),
		persistCh: make(chan struct{}, 1),
		stopCh:    make(chan struct{}),
	}
}

func (mq *MemoryQueue) SetPool(pool *pgxpool.Pool) {
	mq.pool = pool
}

func (mq *MemoryQueue) Produce(topic string, key string, value any) {
	data, err := json.Marshal(value)
	if err != nil {
		mq.logger.Errorw("failed to marshal message value", "topic", topic, "error", err)
		return
	}

	mq.mu.Lock()
	defer mq.mu.Unlock()

	if mq.stopped {
		return
	}

	ts := mq.getOrCreateTopic(topic)

	msg := &Message{
		Seq:   ts.NextSeq,
		Key:   key,
		Value: data,
	}
	ts.NextSeq++

	ts.Messages[ts.Tail] = msg

	ts.Tail = (ts.Tail + 1) % ts.Capacity
	if ts.Tail == ts.Head {
		ts.Head = (ts.Head + 1) % ts.Capacity
	}

	for _, cs := range ts.Consumers {
		select {
		case cs.Ch <- msg:
		default:
		}
	}

	mq.dirty = true
}

func (mq *MemoryQueue) getOrCreateTopic(topic string) *topicState {
	ts, ok := mq.topics[topic]
	if !ok {
		ts = &topicState{
			Messages: make([]*Message, defaultCapacity),
			Capacity: defaultCapacity,
		}
		mq.topics[topic] = ts
	}

	return ts
}

func (mq *MemoryQueue) Close(ctx context.Context) error {
	mq.mu.Lock()
	mq.stopped = true
	close(mq.stopCh)
	mq.mu.Unlock()
	mq.persist(ctx)

	return nil
}

func (mq *MemoryQueue) Reset() {
	mq.mu.Lock()
	mq.stopped = false
	mq.stopCh = make(chan struct{})
	mq.mu.Unlock()
}

func (mq *MemoryQueue) StartPersistLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mq.persist(ctx)
		case <-mq.persistCh:
			mq.persist(ctx)
		case <-mq.stopCh:
			mq.persist(ctx)
			return
		}
	}
}

func (mq *MemoryQueue) SignalPersist() {
	select {
	case mq.persistCh <- struct{}{}:
	default:
	}
}

func serializeTopicMessages(ts *topicState) []*Message {
	if ts.Head == ts.Tail {
		return nil
	}

	count := (ts.Tail - ts.Head + ts.Capacity) % ts.Capacity

	msgs := make([]*Message, 0, count)
	for i := range count {
		idx := (ts.Head + i) % ts.Capacity
		if ts.Messages[idx] != nil {
			msgs = append(msgs, ts.Messages[idx])
		}
	}

	return msgs
}

type persistState struct {
	Updated time.Time                    `json:"updated_at"`
	Topics  map[string]topicSnapshot     `json:"topics"`
	Offsets map[string]map[string]uint64 `json:"offsets"`
}

type topicSnapshot struct {
	Messages []*Message `json:"messages"`
	NextSeq  uint64     `json:"next_seq"`
	Tail     int        `json:"tail"`
}

func (mq *MemoryQueue) buildPersistState() persistState {
	mq.mu.RLock()
	defer mq.mu.RUnlock()

	ps := persistState{
		Updated: time.Now(),
		Topics:  make(map[string]topicSnapshot, len(mq.topics)),
		Offsets: mq.offsets,
	}

	for name, ts := range mq.topics {
		ps.Topics[name] = topicSnapshot{
			Messages: serializeTopicMessages(ts),
			NextSeq:  ts.NextSeq,
			Tail:     ts.Tail,
		}
	}

	return ps
}

func (mq *MemoryQueue) persist(ctx context.Context) {
	mq.mu.Lock()
	if !mq.dirty {
		mq.mu.Unlock()
		return
	}

	mq.dirty = false
	mq.mu.Unlock()

	if mq.pool == nil {
		mq.mu.Lock()
		mq.dirty = true
		mq.mu.Unlock()

		return
	}

	ps := mq.buildPersistState()

	data, err := json.Marshal(ps)
	if err != nil {
		mq.logger.Errorw("failed to marshal queue state", "error", err)
		mq.mu.Lock()
		mq.dirty = true
		mq.mu.Unlock()

		return
	}

	var buf bytes.Buffer

	w := gzip.NewWriter(&buf)
	if _, err := w.Write(data); err != nil {
		_ = w.Close()

		mq.logger.Errorw("failed to compress queue state", "error", err)
		mq.mu.Lock()
		mq.dirty = true
		mq.mu.Unlock()

		return
	}

	_ = w.Close()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	q := db.NewQueries(mq.pool)
	if err := q.UpsertKeyValue(ctx, db.UpsertKeyValueParams{
		Key:   memqueueStateKey,
		Value: buf.Bytes(),
	}); err != nil {
		mq.logger.Errorw("failed to persist queue state", "error", err)
		mq.mu.Lock()
		mq.dirty = true
		mq.mu.Unlock()
	}
}

func (mq *MemoryQueue) Recover(ctx context.Context) {
	if mq.pool == nil {
		mq.logger.Warnw("no database pool set, cannot recover queue state")
		return
	}

	q := db.NewQueries(mq.pool)

	record, err := q.GetKeyValue(ctx, memqueueStateKey)
	if err != nil {
		mq.logger.Infow("no queue state found in database, starting empty")
		return
	}

	if len(record.Value) == 0 {
		return
	}

	zr, err := gzip.NewReader(bytes.NewReader(record.Value))
	if err != nil {
		mq.logger.Errorw("corrupt compressed queue state in database, starting empty", "error", err)
		return
	}

	decompressed, err := io.ReadAll(zr)
	_ = zr.Close()

	if err != nil {
		mq.logger.Errorw("failed to decompress queue state, starting empty", "error", err)
		return
	}

	var ps persistState
	if err := json.Unmarshal(decompressed, &ps); err != nil {
		mq.logger.Errorw("corrupt queue state in database, starting empty", "error", err)
		return
	}

	for name, snap := range ps.Topics {
		ts := &topicState{
			Messages: make([]*Message, defaultCapacity),
			NextSeq:  snap.NextSeq,
			Capacity: defaultCapacity,
			Head:     0,
			Tail:     0,
		}

		for _, msg := range snap.Messages {
			if msg == nil {
				continue
			}

			ts.Messages[ts.Tail] = msg
			ts.Tail++
		}

		ts.Tail %= ts.Capacity

		mq.topics[name] = ts
	}

	mq.offsets = ps.Offsets

	mq.logger.Infow("recovered memory queue state from database", "topics", len(mq.topics))
}
