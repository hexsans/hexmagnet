package configmgr

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hexsans/hexmagnet/internal/protocol/dht"
	"github.com/hexsans/hexmagnet/internal/protocol/metainfo/metainforequester"
	"github.com/hexsans/hexmagnet/internal/servercfg"
	"go.uber.org/zap/zaptest"
)

func TestNewManager(t *testing.T) {
	t.Parallel()

	logger := zaptest.NewLogger(t).Sugar()

	m := NewManager(nil, "", nil, logger)
	if m == nil {
		t.Fatal("expected non-nil manager")
	}

	if snap := m.Get(); snap == nil {
		t.Fatal("expected non-nil snapshot")
	}

	m.Stop()
}

func TestGetStoresInitialSnapshot(t *testing.T) {
	t.Parallel()

	logger := zaptest.NewLogger(t).Sugar()
	initial := &Snapshot{
		DHT: dht.Config{Port: 3334},
	}

	m := NewManager(initial, "", nil, logger)
	defer m.Stop()

	snap := m.Get()
	if snap.DHT.Port != 3334 {
		t.Fatalf("expected port 3334, got %d", snap.DHT.Port)
	}
}

func TestApplyAndGet(t *testing.T) {
	t.Parallel()

	logger := zaptest.NewLogger(t).Sugar()

	m := NewManager(nil, "", nil, logger)
	defer m.Stop()

	snap := &Snapshot{
		DHTRequester: metainforequester.Config{RequestLimit: 100},
		Server:       servercfg.Config{IP: "0.0.0.0"},
	}

	if err := m.Apply(context.Background(), snap); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	got := m.Get()
	if got.DHTRequester.RequestLimit != 100 {
		t.Fatalf("expected RequestLimit 100, got %d", got.DHTRequester.RequestLimit)
	}

	if got.Server.IP != "0.0.0.0" {
		t.Fatalf("expected IP 0.0.0.0, got %s", got.Server.IP)
	}
}

func TestSyncSubscriberCalled(t *testing.T) {
	t.Parallel()

	logger := zaptest.NewLogger(t).Sugar()

	m := NewManager(nil, "", nil, logger)
	defer m.Stop()

	called := make(chan struct{}, 1)

	m.Subscribe(context.Background(), "test_sync", func(_ context.Context, snap *Snapshot) error {
		if snap.DHTRequester.RequestLimit != 50 {
			t.Errorf("expected RequestLimit 50, got %d", snap.DHTRequester.RequestLimit)
		}

		called <- struct{}{}

		return nil
	}, ApplySync)

	snap := &Snapshot{
		DHTRequester: metainforequester.Config{RequestLimit: 50},
	}
	if err := m.Apply(context.Background(), snap); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("sync subscriber was not called")
	}
}

func TestSyncSubscriberErrorPropagates(t *testing.T) {
	t.Parallel()

	logger := zaptest.NewLogger(t).Sugar()

	m := NewManager(nil, "", nil, logger)
	defer m.Stop()

	m.Subscribe(context.Background(), "failing_sync", func(_ context.Context, _ *Snapshot) error {
		return fmt.Errorf("test error")
	}, ApplySync)

	snap := &Snapshot{}

	err := m.Apply(context.Background(), snap)
	if err == nil {
		t.Fatal("expected error from sync subscriber")
	}
}

func TestAsyncSubscriberCalled(t *testing.T) {
	t.Parallel()

	logger := zaptest.NewLogger(t).Sugar()

	m := NewManager(nil, "", nil, logger)
	defer m.Stop()

	called := make(chan struct{}, 1)

	m.Subscribe(context.Background(), "test_async", func(_ context.Context, snap *Snapshot) error {
		if snap.DHTRequester.HashDiscoverLimit != 20 {
			t.Errorf("expected HashDiscoverLimit 20, got %d", snap.DHTRequester.HashDiscoverLimit)
		}

		called <- struct{}{}

		return nil
	}, ApplyAsync)

	snap := &Snapshot{
		DHTRequester: metainforequester.Config{HashDiscoverLimit: 20},
	}
	if err := m.Apply(context.Background(), snap); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("async subscriber was not called")
	}
}

func TestAsyncSubscriberSkipsDuplicate(t *testing.T) {
	t.Parallel()

	logger := zaptest.NewLogger(t).Sugar()

	m := NewManager(nil, "", nil, logger)
	defer m.Stop()

	var callCount int

	called := make(chan struct{}, 10)

	m.Subscribe(context.Background(), "test_dedup", func(_ context.Context, _ *Snapshot) error {
		callCount++

		called <- struct{}{}

		return nil
	}, ApplyAsync)

	snap := &Snapshot{DHTRequester: metainforequester.Config{RequestLimit: 10}}
	if err := m.Apply(context.Background(), snap); err != nil {
		t.Fatalf("first Apply failed: %v", err)
	}

	<-called

	if err := m.Apply(context.Background(), snap); err != nil {
		t.Fatalf("second Apply failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	if callCount != 1 {
		t.Fatalf("expected 1 call (dedup), got %d", callCount)
	}
}

func TestAsyncSubscriberDifferentSnapshots(t *testing.T) {
	t.Parallel()

	logger := zaptest.NewLogger(t).Sugar()

	m := NewManager(nil, "", nil, logger)
	defer m.Stop()

	var callCount int

	called := make(chan struct{}, 10)

	m.Subscribe(context.Background(), "test_multiple", func(_ context.Context, _ *Snapshot) error {
		callCount++

		called <- struct{}{}

		return nil
	}, ApplyAsync)

	snap1 := &Snapshot{DHTRequester: metainforequester.Config{RequestLimit: 10}}
	if err := m.Apply(context.Background(), snap1); err != nil {
		t.Fatalf("first Apply failed: %v", err)
	}

	<-called

	snap2 := &Snapshot{DHTRequester: metainforequester.Config{RequestLimit: 20}}
	if err := m.Apply(context.Background(), snap2); err != nil {
		t.Fatalf("second Apply failed: %v", err)
	}

	<-called

	if callCount != 2 {
		t.Fatalf("expected 2 calls, got %d", callCount)
	}
}

func TestWriteFnCalled(t *testing.T) {
	t.Parallel()

	logger := zaptest.NewLogger(t).Sugar()

	var writtenSnap *Snapshot

	writeFn := func(_ string, snap *Snapshot) error {
		writtenSnap = snap
		return nil
	}

	m := NewManager(nil, "/tmp/test.yaml", writeFn, logger)
	defer m.Stop()

	snap := &Snapshot{DHTRequester: metainforequester.Config{RequestLimit: 75}}
	if err := m.Apply(context.Background(), snap); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	if writtenSnap == nil {
		t.Fatal("writeFn was not called")
	}

	if writtenSnap.DHTRequester.RequestLimit != 75 {
		t.Fatalf("expected RequestLimit 75, got %d", writtenSnap.DHTRequester.RequestLimit)
	}
}

func TestSyncFailureLeavesSnapshotAndFileUntouched(t *testing.T) {
	t.Parallel()

	logger := zaptest.NewLogger(t).Sugar()
	initial := &Snapshot{DHTRequester: metainforequester.Config{RequestLimit: 1}}

	writeCalls := 0
	writeFn := func(string, *Snapshot) error {
		writeCalls++
		return nil
	}

	m := NewManager(initial, "/tmp/test.yaml", writeFn, logger)
	defer m.Stop()

	m.Subscribe(context.Background(), "failing_sync", func(_ context.Context, _ *Snapshot) error {
		return fmt.Errorf("boom")
	}, ApplySync)

	newSnap := &Snapshot{DHTRequester: metainforequester.Config{RequestLimit: 2}}

	err := m.Apply(context.Background(), newSnap)
	if err == nil {
		t.Fatal("expected error from sync subscriber")
	}

	if got := m.Get(); got.DHTRequester.RequestLimit != 1 {
		t.Fatalf("snapshot must stay on old value, got limit %d", got.DHTRequester.RequestLimit)
	}

	if writeCalls != 0 {
		t.Fatalf("writeFn must not be called when a sync subscriber fails, got %d calls", writeCalls)
	}
}

func TestWriteFailureLeavesSnapshotUntouched(t *testing.T) {
	t.Parallel()

	logger := zaptest.NewLogger(t).Sugar()
	initial := &Snapshot{DHTRequester: metainforequester.Config{RequestLimit: 1}}

	m := NewManager(initial, "/tmp/test.yaml", func(string, *Snapshot) error {
		return fmt.Errorf("disk full")
	}, logger)
	defer m.Stop()

	newSnap := &Snapshot{DHTRequester: metainforequester.Config{RequestLimit: 2}}

	err := m.Apply(context.Background(), newSnap)
	if err == nil {
		t.Fatal("expected error from writeFn")
	}

	if got := m.Get(); got.DHTRequester.RequestLimit != 1 {
		t.Fatalf("snapshot must stay on old value, got limit %d", got.DHTRequester.RequestLimit)
	}
}

func TestApplyOrderSyncThenWriteThenAsync(t *testing.T) {
	t.Parallel()

	logger := zaptest.NewLogger(t).Sugar()

	m := NewManager(nil, "/tmp/test.yaml", nil, logger)
	defer m.Stop()

	var events []string

	event := func(e string) {
		events = append(events, e)
	}

	writeFn := func(string, *Snapshot) error {
		event("write")
		return nil
	}
	m.filePath = "/tmp/test.yaml"
	m.writeFn = writeFn

	m.Subscribe(context.Background(), "sync", func(_ context.Context, _ *Snapshot) error {
		event("sync")
		return nil
	}, ApplySync)

	asyncDone := make(chan struct{}, 1)

	m.Subscribe(context.Background(), "async", func(_ context.Context, _ *Snapshot) error {
		event("async")

		asyncDone <- struct{}{}

		return nil
	}, ApplyAsync)

	if err := m.Apply(context.Background(), &Snapshot{}); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	select {
	case <-asyncDone:
	case <-time.After(time.Second):
		t.Fatal("async subscriber was never called")
	}

	if len(events) != 3 || events[0] != "sync" || events[1] != "write" || events[2] != "async" {
		t.Fatalf("expected order [sync write async], got %v", events)
	}
}

func TestAsyncChannelFullReplacesPending(t *testing.T) {
	t.Parallel()

	logger := zaptest.NewLogger(t).Sugar()

	m := NewManager(nil, "", nil, logger)
	defer m.Stop()

	blocked := make(chan struct{})
	seen := make(chan *Snapshot, 10)

	m.Subscribe(context.Background(), "slow_sub", func(_ context.Context, snap *Snapshot) error {
		seen <- snap

		<-blocked

		return nil
	}, ApplyAsync)

	snap := &Snapshot{DHTRequester: metainforequester.Config{RequestLimit: 1}}
	if err := m.Apply(context.Background(), snap); err != nil {
		t.Fatalf("first Apply should succeed: %v", err)
	}

	<-seen

	snap2 := &Snapshot{DHTRequester: metainforequester.Config{RequestLimit: 2}}
	if err := m.Apply(context.Background(), snap2); err != nil {
		t.Fatalf("second Apply should succeed: %v", err)
	}

	// Subscriber is still busy with the first snapshot; a third Apply should
	// replace the pending snapshot instead of dropping the update.
	snap3 := &Snapshot{DHTRequester: metainforequester.Config{RequestLimit: 3}}
	if err := m.Apply(context.Background(), snap3); err != nil {
		t.Fatalf("third Apply should succeed: %v", err)
	}

	close(blocked)

	select {
	case got := <-seen:
		if got.DHTRequester.RequestLimit != 3 {
			t.Fatalf("expected latest snapshot (limit 3) to be applied, got limit %d", got.DHTRequester.RequestLimit)
		}
	case <-time.After(time.Second):
		t.Fatal("latest pending snapshot was never applied")
	}
}
