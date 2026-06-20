package resolvers

import (
	"context"
	"testing"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/hexsans/hexmagnet/internal/classifier"
	"github.com/hexsans/hexmagnet/internal/gql/gqlmodel"
	"github.com/hexsans/hexmagnet/internal/gql/gqlmodel/gen"
	"github.com/hexsans/hexmagnet/internal/health"
	"github.com/hexsans/hexmagnet/internal/metrics/queuemetrics"
	"github.com/hexsans/hexmagnet/internal/model"
	"github.com/hexsans/hexmagnet/internal/processor"
	"github.com/hexsans/hexmagnet/internal/protocol"
	"github.com/hexsans/hexmagnet/internal/testutil"
	"github.com/hexsans/hexmagnet/internal/version"
	"github.com/hexsans/hexmagnet/internal/worker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type testShutdowner struct{}

func (testShutdowner) Shutdown(_ ...fx.ShutdownOption) error { return nil }

type mockHealthChecker struct {
	mock.Mock
}

func (*mockHealthChecker) Start()                            {}
func (*mockHealthChecker) Stop()                             {}
func (*mockHealthChecker) GetRunningPeriodicCheckCount() int { return 0 }
func (*mockHealthChecker) IsStarted() bool                   { return true }
func (*mockHealthChecker) StartedAt() time.Time              { return time.Time{} }
func (m *mockHealthChecker) Check(ctx context.Context) health.CheckerResult {
	return m.Called(ctx).Get(0).(health.CheckerResult)
}

type mockProcessor struct {
	mock.Mock
}

func (m *mockProcessor) Process(ctx context.Context, params processor.MessageParams) error {
	return m.Called(ctx, params).Error(0)
}

func (m *mockProcessor) UpdateTorrentFilter(cfg classifier.TorrentFilterConfig) error {
	return m.Called(cfg).Error(0)
}

func (m *mockProcessor) SwapRunner(runner classifier.Runner) {
	m.Called(runner)
}

func TestMutation_Getter(t *testing.T) {
	t.Parallel()

	r := &Resolver{}
	m := r.Mutation()
	assert.NotNil(t, m)
}

func TestTorrentMutation_Getter(t *testing.T) {
	t.Parallel()

	r := &Resolver{}
	tm := r.TorrentMutation()
	assert.NotNil(t, tm)
}

func TestTorrentMutation_ReturnsMutation(t *testing.T) {
	t.Parallel()

	r := &Resolver{}
	m := &mutationResolver{r}
	tm, err := m.Torrent(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, tm)
}

func TestReprocess(t *testing.T) {
	t.Parallel()

	mockProc := new(mockProcessor)
	mockProc.On("Process", mock.Anything, mock.Anything).Return(nil)

	r := &Resolver{Processor: mockProc}
	tm := &torrentMutationResolver{r}

	input := gen.TorrentReprocessInput{
		InfoHashes: []protocol.ID{
			testutil.MustParseID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		},
	}

	result, err := tm.Reprocess(context.Background(), &gqlmodel.TorrentMutation{}, input)
	require.NoError(t, err)
	assert.Nil(t, result)
	mockProc.AssertExpectations(t)
}

func TestReprocess_ClassifierRematch(t *testing.T) {
	t.Parallel()

	mockProc := new(mockProcessor)
	rematch := true

	mockProc.On("Process", mock.Anything, mock.MatchedBy(func(params processor.MessageParams) bool {
		return params.ClassifyMode == processor.ClassifyModeRematch
	})).Return(nil)

	r := &Resolver{Processor: mockProc}
	tm := &torrentMutationResolver{r}

	input := gen.TorrentReprocessInput{
		InfoHashes:        []protocol.ID{testutil.MustParseID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")},
		ClassifierRematch: graphql.OmittableOf[*bool](&rematch),
	}

	_, err := tm.Reprocess(context.Background(), &gqlmodel.TorrentMutation{}, input)
	require.NoError(t, err)
	mockProc.AssertExpectations(t)
}

func TestReprocess_ContentType(t *testing.T) {
	t.Parallel()

	mockProc := new(mockProcessor)
	mockProc.On("Process", mock.Anything, mock.Anything).Return(nil)

	r := &Resolver{Processor: mockProc}
	tm := &torrentMutationResolver{r}

	ct := model.ContentTypeMovie
	input := gen.TorrentReprocessInput{
		InfoHashes:  []protocol.ID{testutil.MustParseID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")},
		ContentType: graphql.OmittableOf[*model.ContentType](&ct),
	}

	_, err := tm.Reprocess(context.Background(), &gqlmodel.TorrentMutation{}, input)
	require.NoError(t, err)
	mockProc.AssertExpectations(t)
}

func TestQueueMetricsBucket_Getter(t *testing.T) {
	t.Parallel()

	r := &Resolver{}
	qb := r.QueueMetricsBucket()
	assert.NotNil(t, qb)
}

func TestStatus(t *testing.T) {
	t.Parallel()

	r := &Resolver{}
	qb := &queueMetricsBucketResolver{r}

	tests := []struct {
		status string
		want   gen.QueueJobStatus
	}{
		{"pending", gen.QueueJobStatusPending},
		{"processed", gen.QueueJobStatusProcessed},
		{"failed", gen.QueueJobStatusFailed},
		{"retry", gen.QueueJobStatusRetry},
		{"unknown", gen.QueueJobStatus("unknown")},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			t.Parallel()

			bucket := &queuemetrics.Bucket{Status: tt.status}
			got, err := qb.Status(context.Background(), bucket)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestVersion(t *testing.T) {
	t.Parallel()

	r := &Resolver{}
	q := &queryResolver{r}

	ver, err := q.Version(context.Background())
	require.NoError(t, err)
	assert.Equal(t, version.GitTag, ver)
}

func TestWorkers(t *testing.T) {
	t.Parallel()

	w1 := worker.NewWorker("worker1", fx.Hook{})
	w2 := worker.NewWorker("worker2", fx.Hook{})

	regResult, err := worker.NewRegistry(worker.RegistryParams{
		Shutdowner: testShutdowner{},
		Workers:    []worker.Worker{w1, w2},
		Logger:     zap.NewNop().Sugar(),
	})
	require.NoError(t, err)

	r := &Resolver{Workers: regResult.Registry}
	q := &queryResolver{r}

	result, err := q.Workers(context.Background())
	require.NoError(t, err)

	require.Len(t, result.ListAll.Workers, 2)
	assert.Equal(t, "worker1", result.ListAll.Workers[0].Key)
	assert.False(t, result.ListAll.Workers[0].Started)
	assert.Equal(t, "worker2", result.ListAll.Workers[1].Key)
	assert.False(t, result.ListAll.Workers[1].Started)
}

func TestHealth(t *testing.T) {
	t.Parallel()

	now := time.Now()
	checker := new(mockHealthChecker)
	checker.On("Check", mock.Anything).Return(health.CheckerResult{
		Status: health.StatusUp,
		Details: map[string]health.CheckResult{
			"db": {
				Status:    health.StatusUp,
				Timestamp: now,
			},
		},
	})

	r := &Resolver{Checker: checker}
	q := &queryResolver{r}

	result, err := q.Health(context.Background())
	require.NoError(t, err)

	assert.Equal(t, gen.HealthStatusUp, result.Status)
	require.Len(t, result.Checks, 1)
	assert.Equal(t, "db", result.Checks[0].Key)
	assert.Equal(t, gen.HealthStatusUp, result.Checks[0].Status)
	assert.Equal(t, now, result.Checks[0].Timestamp)
	assert.Nil(t, result.Checks[0].Error)
	checker.AssertExpectations(t)
}

func TestHealth_SkippedInactive(t *testing.T) {
	t.Parallel()

	checker := new(mockHealthChecker)
	checker.On("Check", mock.Anything).Return(health.CheckerResult{
		Status: health.StatusUp,
		Details: map[string]health.CheckResult{
			"active":   {Status: health.StatusUp, Timestamp: time.Now()},
			"inactive": {Status: health.StatusInactive, Timestamp: time.Now()},
		},
	})

	r := &Resolver{Checker: checker}
	q := &queryResolver{r}

	result, err := q.Health(context.Background())
	require.NoError(t, err)
	require.Len(t, result.Checks, 1)
	assert.Equal(t, "active", result.Checks[0].Key)
	checker.AssertExpectations(t)
}

func TestHealth_WithError(t *testing.T) {
	t.Parallel()

	checker := new(mockHealthChecker)
	checker.On("Check", mock.Anything).Return(health.CheckerResult{
		Status: health.StatusDown,
		Details: map[string]health.CheckResult{
			"db": {
				Status:    health.StatusDown,
				Timestamp: time.Now(),
				Error:     assert.AnError,
			},
		},
	})

	r := &Resolver{Checker: checker}
	q := &queryResolver{r}

	result, err := q.Health(context.Background())
	require.NoError(t, err)
	assert.Equal(t, gen.HealthStatusDown, result.Status)
	require.Len(t, result.Checks, 1)
	require.NotNil(t, result.Checks[0].Error)
	assert.Equal(t, assert.AnError.Error(), *result.Checks[0].Error)
	checker.AssertExpectations(t)
}

func TestHealth_AllStatusValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input health.AvailabilityStatus
		want  gen.HealthStatus
	}{
		{health.StatusUp, gen.HealthStatusUp},
		{health.StatusDown, gen.HealthStatusDown},
		{health.StatusInactive, gen.HealthStatusInactive},
		{health.StatusUnknown, gen.HealthStatusUnknown},
		{"custom", gen.HealthStatusUnknown},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			t.Parallel()

			checker := new(mockHealthChecker)
			checker.On("Check", mock.Anything).Return(health.CheckerResult{
				Status: tt.input,
				Details: map[string]health.CheckResult{
					"check": {Status: tt.input, Timestamp: time.Now()},
				},
			})

			r := &Resolver{Checker: checker}
			q := &queryResolver{r}

			result, err := q.Health(context.Background())
			require.NoError(t, err)
			assert.Equal(t, tt.want, result.Status)
			checker.AssertExpectations(t)
		})
	}
}

func TestDhtCrawler(t *testing.T) {
	t.Parallel()

	r := &Resolver{}
	q := &queryResolver{r}

	status, err := q.DhtCrawler(context.Background())
	require.NoError(t, err)
	assert.Nil(t, status.Runtime)
	assert.Nil(t, status.DB)
}

func TestQuery_Torrent(t *testing.T) {
	t.Parallel()

	r := &Resolver{}
	q := &queryResolver{r}

	tq, err := q.Torrent(context.Background())
	require.NoError(t, err)
	assert.Nil(t, tq.DB)
	assert.Nil(t, tq.Search)
	assert.Nil(t, tq.TorrentMetricsClient)
}

func TestQuery_TorrentSearch(t *testing.T) {
	t.Parallel()

	r := &Resolver{Logger: zap.NewNop().Sugar()}
	q := &queryResolver{r}

	ts, err := q.TorrentSearch(context.Background())
	require.NoError(t, err)
	assert.Nil(t, ts.Svc)
	assert.NotNil(t, ts.Logger)
}

func TestQuery_Queue(t *testing.T) {
	t.Parallel()

	r := &Resolver{}
	q := &queryResolver{r}

	qq, err := q.Queue(context.Background())
	require.NoError(t, err)
	assert.Nil(t, qq.QueueManager)
}
