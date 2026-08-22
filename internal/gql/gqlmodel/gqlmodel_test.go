package gqlmodel

import (
	"context"
	"testing"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/hexsans/hexmagnet/internal/dhtcrawler"
	"github.com/hexsans/hexmagnet/internal/gql/gqlmodel/gen"
	"github.com/hexsans/hexmagnet/internal/metrics/torrentmetrics"
	"github.com/hexsans/hexmagnet/internal/model"
	"github.com/hexsans/hexmagnet/internal/queue"
	dbsearch "github.com/hexsans/hexmagnet/internal/search"
	"github.com/hexsans/hexmagnet/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- nilToZero ---

func TestNilToZero_Int(t *testing.T) {
	t.Parallel()

	v := 42
	assert.Equal(t, 42, nilToZero(&v))
	assert.Equal(t, 0, nilToZero[int](nil))
}

func TestNilToZero_String(t *testing.T) {
	t.Parallel()

	s := "hello"
	assert.Equal(t, "hello", nilToZero(&s))
	assert.Empty(t, nilToZero[string](nil))
}

func TestNilToZero_Struct(t *testing.T) {
	t.Parallel()

	type s struct{ A int }

	v := s{A: 10}
	assert.Equal(t, s{A: 10}, nilToZero(&v))
	assert.Equal(t, s{}, nilToZero[s](nil))
}

// --- aggs helpers ---

func TestContentTypeAggs_Nil(t *testing.T) {
	t.Parallel()

	result, err := contentTypeAggs(nil)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestContentTypeAggs_Empty(t *testing.T) {
	t.Parallel()

	result, err := contentTypeAggs(dbsearch.AggregationItems{})
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestContentTypeAggs_Valid(t *testing.T) {
	t.Parallel()

	items := dbsearch.AggregationItems{
		"movie": {Count: 10, Label: "Movie", IsEstimate: false},
	}
	result, err := contentTypeAggs(items)
	require.NoError(t, err)
	require.Len(t, result, 1)

	movie := model.ContentType("movie")
	assert.Equal(t, &movie, result[0].Value)
	assert.Equal(t, "Movie", result[0].Label)
	assert.Equal(t, 10, result[0].Count)
}

func TestContentTypeAggs_WithNull(t *testing.T) {
	t.Parallel()

	items := dbsearch.AggregationItems{
		"null": {Count: 5, Label: "Unknown", IsEstimate: true},
	}
	result, err := contentTypeAggs(items)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Nil(t, result[0].Value)
	assert.Equal(t, "Unknown", result[0].Label)
	assert.Equal(t, 5, result[0].Count)
	assert.True(t, result[0].IsEstimate)
}

func TestContentTypeAggs_Invalid(t *testing.T) {
	t.Parallel()

	items := dbsearch.AggregationItems{
		"invalid_type": {Count: 1, Label: "Invalid"},
	}
	_, err := contentTypeAggs(items)
	require.Error(t, err)
}

func TestTorrentFileTypeAggs_Nil(t *testing.T) {
	t.Parallel()

	result, err := torrentFileTypeAggs(nil)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestTorrentFileTypeAggs_Valid(t *testing.T) {
	t.Parallel()

	items := dbsearch.AggregationItems{
		"video": {Count: 20, Label: "Video", IsEstimate: false},
	}
	result, err := torrentFileTypeAggs(items)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, model.FileType("video"), result[0].Value)
	assert.Equal(t, "Video", result[0].Label)
	assert.Equal(t, 20, result[0].Count)
}

func TestTorrentFileTypeAggs_Invalid(t *testing.T) {
	t.Parallel()

	items := dbsearch.AggregationItems{
		"not_a_file_type": {Count: 1, Label: "Unknown"},
	}
	_, err := torrentFileTypeAggs(items)
	require.Error(t, err)
}

// --- DhtCrawlerStatus ---

func TestDhtCrawlerStatus_Active(t *testing.T) {
	t.Parallel()

	r := dhtcrawler.NewRuntime()
	s := DhtCrawlerStatus{Runtime: r, DB: nil}

	r.Active.Set(true)
	active, err := s.Active(context.Background())
	require.NoError(t, err)
	assert.True(t, active)

	r.Active.Set(false)
	active, err = s.Active(context.Background())
	require.NoError(t, err)
	assert.False(t, active)
}

func TestDhtCrawlerStatus_PeersDiscovered(t *testing.T) {
	t.Parallel()

	r := dhtcrawler.NewRuntime()
	s := DhtCrawlerStatus{Runtime: r}

	r.PeersDiscovered.Set(uint64(100))
	count, err := s.PeersDiscovered(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint64(100), count)

	r.PeersDiscovered.Set(uint64(0))
	count, err = s.PeersDiscovered(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint64(0), count)
}

func TestDhtCrawlerStatus_PeersConnected(t *testing.T) {
	t.Parallel()

	r := dhtcrawler.NewRuntime()
	s := DhtCrawlerStatus{Runtime: r}

	r.PeersConnected.Set(uint64(50))
	count, err := s.PeersConnected(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint64(50), count)
}

func TestDhtCrawlerStatus_StartedAt_Zero(t *testing.T) {
	t.Parallel()

	r := dhtcrawler.NewRuntime()
	s := DhtCrawlerStatus{Runtime: r}

	started, err := s.StartedAt(context.Background())
	require.NoError(t, err)
	assert.Nil(t, started)
}

func TestDhtCrawlerStatus_StartedAt_NonZero(t *testing.T) {
	t.Parallel()

	r := dhtcrawler.NewRuntime()
	s := DhtCrawlerStatus{Runtime: r}

	now := time.Now()
	r.StartedAt.Set(now)
	started, err := s.StartedAt(context.Background())
	require.NoError(t, err)
	require.NotNil(t, started)
	assert.True(t, started.Equal(now))
}

func TestDhtCrawlerStatus_Uptime_Initial(t *testing.T) {
	t.Parallel()

	r := dhtcrawler.NewRuntime()
	s := DhtCrawlerStatus{Runtime: r}

	uptime, err := s.Uptime(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint64(0), uptime)
}

func TestDhtCrawlerStatus_Uptime_WithOffset(t *testing.T) {
	t.Parallel()

	r := dhtcrawler.NewRuntime()
	s := DhtCrawlerStatus{Runtime: r}

	r.SetUptimeOffset(10 * time.Second)

	uptime, err := s.Uptime(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint64(10), uptime)
}

func TestDhtCrawlerStatus_RecentActivity_Empty(t *testing.T) {
	t.Parallel()

	r := dhtcrawler.NewRuntime()
	s := DhtCrawlerStatus{Runtime: r}

	entries, err := s.RecentActivity(context.Background())
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestDhtCrawlerStatus_RecentActivity_WithEntries(t *testing.T) {
	t.Parallel()

	r := dhtcrawler.NewRuntime()
	s := DhtCrawlerStatus{Runtime: r}

	r.PushActivity("announce", "announced to network")
	r.PushActivity("ping", "pinged node")

	entries, err := s.RecentActivity(context.Background())
	require.NoError(t, err)
	require.Len(t, entries, 2)
	assert.Equal(t, "ping", entries[0].Type)
	assert.Equal(t, "pinged node", entries[0].Message)
	assert.Equal(t, "announce", entries[1].Type)
	assert.Equal(t, "announced to network", entries[1].Message)
}

// --- torrentSearchRowToGQL ---

func int32Ptr(v int32) *int32 { return &v }

func timePtr(t time.Time) *time.Time { return &t }

func TestTorrentSearchRowToGQL_Basic(t *testing.T) {
	t.Parallel()

	ct := "movie"
	row := dbsearch.TorrentSearchRow{
		InfoHash:         testutil.ValidHash(),
		ContentType:      &ct,
		Seeders:          int32Ptr(10),
		Leechers:         int32Ptr(5),
		TorrentName:      "Test Torrent",
		Size:             1024,
		ContentTitle:     testutil.StrPtr("Test Content"),
		ContentOverview:  testutil.StrPtr("An overview"),
		ContentCreatedAt: timePtr(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
		FilesCount:       int32Ptr(3),
		CreatedAt:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:        time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
	}

	result := torrentSearchRowToGQL(row)

	assert.Equal(t, "Test Content", result.Title)
	assert.Equal(t, result.InfoHash.String(), testutil.ValidHash())
	assert.Equal(t, uint(10), result.Seeders.Uint)
	assert.True(t, result.Seeders.Valid)
	assert.Equal(t, uint(5), result.Leechers.Uint)
	assert.True(t, result.Leechers.Valid)
	assert.True(t, result.ContentType.Valid)
	assert.Equal(t, model.ContentType("movie"), result.ContentType.ContentType)
	require.NotNil(t, result.Content)
	assert.Equal(t, "Test Content", result.Content.Title)
	assert.Equal(t, "An overview", result.Content.Overview.String)
	assert.True(t, result.Content.Overview.Valid)
}

func TestTorrentSearchRowToGQL_NoContent(t *testing.T) {
	t.Parallel()

	row := dbsearch.TorrentSearchRow{
		InfoHash:    testutil.ValidHash(),
		TorrentName: "Just A Torrent",
		Size:        512,
		FilesCount:  int32Ptr(1),
		CreatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	result := torrentSearchRowToGQL(row)

	assert.Equal(t, "Just A Torrent", result.Title)
	assert.Nil(t, result.Content)
	assert.False(t, result.ContentType.Valid)
	assert.False(t, result.Seeders.Valid)
	assert.False(t, result.Leechers.Valid)
}

func TestTorrentSearchRowToGQL_WithLanguages(t *testing.T) {
	t.Parallel()

	row := dbsearch.TorrentSearchRow{
		InfoHash:    testutil.ValidHash(),
		TorrentName: "Multi Lang Torrent",
		Size:        256,
		Languages:   []byte(`["en","fr"]`),
		FilesCount:  int32Ptr(2),
		CreatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	result := torrentSearchRowToGQL(row)
	assert.Contains(t, result.Languages, model.Language("en"))
	assert.Contains(t, result.Languages, model.Language("fr"))
}

func TestTorrentSearchRowToGQL_ContentSourceAndID(t *testing.T) {
	t.Parallel()

	cs := "tmdb"
	cid := "12345"
	row := dbsearch.TorrentSearchRow{
		InfoHash:      testutil.ValidHash(),
		ContentSource: &cs,
		ContentID:     &cid,
		TorrentName:   "Content Ref Torrent",
		Size:          100,
		FilesCount:    int32Ptr(1),
		CreatedAt:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	result := torrentSearchRowToGQL(row)
	assert.True(t, result.ContentSource.Valid)
	assert.Equal(t, "tmdb", result.ContentSource.String)
	assert.True(t, result.ContentID.Valid)
	assert.Equal(t, "12345", result.ContentID.String)
}

// --- fromInt32PtrNull ---

func TestFromInt32PtrNull_Nil(t *testing.T) {
	t.Parallel()

	result := fromInt32PtrNull(nil)
	assert.False(t, result.Valid)
	assert.Equal(t, uint(0), result.Uint)
}

func TestFromInt32PtrNull_Value(t *testing.T) {
	t.Parallel()

	v := int32(42)
	result := fromInt32PtrNull(&v)
	assert.True(t, result.Valid)
	assert.Equal(t, uint(42), result.Uint)
}

func TestFromInt32PtrNull_Zero(t *testing.T) {
	t.Parallel()

	v := int32(0)
	result := fromInt32PtrNull(&v)
	assert.True(t, result.Valid)
	assert.Equal(t, uint(0), result.Uint)
}

// --- nullStrPtr ---

func TestNullStrPtr_Nil(t *testing.T) {
	t.Parallel()

	result := nullStrPtr(nil)
	assert.False(t, result.Valid)
	assert.Empty(t, result.String)
}

func TestNullStrPtr_Value(t *testing.T) {
	t.Parallel()

	s := "hello"
	result := nullStrPtr(&s)
	assert.True(t, result.Valid)
	assert.Equal(t, "hello", result.String)
}

// --- timePtrOrZero ---

func TestTimePtrOrZero_Nil(t *testing.T) {
	t.Parallel()
	assert.True(t, timePtrOrZero(nil).IsZero())
}

func TestTimePtrOrZero_Value(t *testing.T) {
	t.Parallel()

	now := time.Now()
	assert.Equal(t, now, timePtrOrZero(&now))
}

// --- QueueQuery Metrics (with mock) ---

type mockQueueManager struct {
	mock.Mock
}

func (m *mockQueueManager) Metrics(ctx context.Context, query queue.MetricsQuery) (queue.MetricsResult, error) {
	args := m.Called(ctx, query)
	return args.Get(0).(queue.MetricsResult), args.Error(1)
}

func (m *mockQueueManager) Jobs(ctx context.Context, query queue.JobsQuery) (queue.JobsResult, error) {
	args := m.Called(ctx, query)
	return args.Get(0).(queue.JobsResult), args.Error(1)
}

func (m *mockQueueManager) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestQueueQuery_Metrics_Minute(t *testing.T) {
	t.Parallel()

	mgr := new(mockQueueManager)
	q := QueueQuery{QueueManager: mgr}

	expectedResult := queue.MetricsResult{
		Buckets: []queue.MetricsBucket{
			{Queue: "test", Status: "done", Count: 5},
		},
	}
	mgr.On("Metrics", mock.Anything, mock.MatchedBy(func(q queue.MetricsQuery) bool {
		return q.BucketDuration == time.Minute
	})).Return(expectedResult, nil)

	input := gen.QueueMetricsQueryInput{
		BucketDuration: gen.MetricsBucketDurationMinute,
	}
	result, err := q.Metrics(context.Background(), input)
	require.NoError(t, err)
	require.Len(t, result.Buckets, 1)
	assert.Equal(t, "test", result.Buckets[0].Queue)
	mgr.AssertExpectations(t)
}

func TestQueueQuery_Metrics_Hour(t *testing.T) {
	t.Parallel()

	mgr := new(mockQueueManager)
	q := QueueQuery{QueueManager: mgr}

	expectedResult := queue.MetricsResult{Buckets: nil}
	mgr.On("Metrics", mock.Anything, mock.MatchedBy(func(q queue.MetricsQuery) bool {
		return q.BucketDuration == time.Hour
	})).Return(expectedResult, nil)

	input := gen.QueueMetricsQueryInput{
		BucketDuration: gen.MetricsBucketDurationHour,
	}
	result, err := q.Metrics(context.Background(), input)
	require.NoError(t, err)
	assert.Empty(t, result.Buckets)
	mgr.AssertExpectations(t)
}

func TestQueueQuery_Metrics_Day(t *testing.T) {
	t.Parallel()

	mgr := new(mockQueueManager)
	q := QueueQuery{QueueManager: mgr}

	expectedResult := queue.MetricsResult{Buckets: []queue.MetricsBucket{}}
	mgr.On("Metrics", mock.Anything, mock.MatchedBy(func(q queue.MetricsQuery) bool {
		return q.BucketDuration == 24*time.Hour
	})).Return(expectedResult, nil)

	input := gen.QueueMetricsQueryInput{
		BucketDuration: gen.MetricsBucketDurationDay,
	}
	_, err := q.Metrics(context.Background(), input)
	require.NoError(t, err)
	mgr.AssertExpectations(t)
}

func TestQueueQuery_Metrics_WithFilters(t *testing.T) {
	t.Parallel()

	mgr := new(mockQueueManager)
	q := QueueQuery{QueueManager: mgr}

	now := time.Now()

	mgr.On("Metrics", mock.Anything, mock.MatchedBy(func(qq queue.MetricsQuery) bool {
		return len(qq.Queues) == 1 && qq.Queues[0] == "q1" &&
			len(qq.Statuses) == 1 && qq.Statuses[0] == "done" &&
			!qq.StartTime.IsZero() && !qq.EndTime.IsZero()
	})).Return(queue.MetricsResult{}, nil)

	input := gen.QueueMetricsQueryInput{
		BucketDuration: gen.MetricsBucketDurationMinute,
		Queues:         graphql.OmittableOf([]string{"q1"}),
		Statuses:       graphql.OmittableOf([]gen.QueueJobStatus{gen.QueueJobStatus("done")}),
		StartTime:      graphql.OmittableOf(&now),
		EndTime:        graphql.OmittableOf(&now),
	}
	_, err := q.Metrics(context.Background(), input)
	require.NoError(t, err)
	mgr.AssertExpectations(t)
}

func TestQueueQuery_Metrics_Error(t *testing.T) {
	t.Parallel()

	mgr := new(mockQueueManager)
	q := QueueQuery{QueueManager: mgr}

	mgr.On("Metrics", mock.Anything, mock.Anything).Return(queue.MetricsResult{}, assert.AnError)

	input := gen.QueueMetricsQueryInput{
		BucketDuration: gen.MetricsBucketDurationMinute,
	}
	_, err := q.Metrics(context.Background(), input)
	require.Error(t, err)
	mgr.AssertExpectations(t)
}

// --- TorrentQuery Metrics (with mock) ---

type mockTorrentMetricsClient struct {
	mock.Mock
}

func (m *mockTorrentMetricsClient) Request(ctx context.Context, req torrentmetrics.Request) ([]torrentmetrics.Bucket, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).([]torrentmetrics.Bucket), args.Error(1)
}

func TestTorrentQuery_Metrics_Minute(t *testing.T) {
	t.Parallel()

	mc := new(mockTorrentMetricsClient)
	tq := TorrentQuery{TorrentMetricsClient: mc}

	mc.On("Request", mock.Anything, mock.MatchedBy(func(r torrentmetrics.Request) bool {
		return r.BucketDuration == "minute"
	})).Return([]torrentmetrics.Bucket{}, nil)

	input := gen.TorrentMetricsQueryInput{
		BucketDuration: gen.MetricsBucketDurationMinute,
	}
	_, err := tq.Metrics(context.Background(), input)
	require.NoError(t, err)
	mc.AssertExpectations(t)
}

func TestTorrentQuery_Metrics_Hour(t *testing.T) {
	t.Parallel()

	mc := new(mockTorrentMetricsClient)
	tq := TorrentQuery{TorrentMetricsClient: mc}

	mc.On("Request", mock.Anything, mock.MatchedBy(func(r torrentmetrics.Request) bool {
		return r.BucketDuration == "hour"
	})).Return([]torrentmetrics.Bucket{}, nil)

	input := gen.TorrentMetricsQueryInput{
		BucketDuration: gen.MetricsBucketDurationHour,
	}
	_, err := tq.Metrics(context.Background(), input)
	require.NoError(t, err)
	mc.AssertExpectations(t)
}

func TestTorrentQuery_Metrics_Day(t *testing.T) {
	t.Parallel()

	mc := new(mockTorrentMetricsClient)
	tq := TorrentQuery{TorrentMetricsClient: mc}

	mc.On("Request", mock.Anything, mock.MatchedBy(func(r torrentmetrics.Request) bool {
		return r.BucketDuration == "day"
	})).Return([]torrentmetrics.Bucket{}, nil)

	input := gen.TorrentMetricsQueryInput{
		BucketDuration: gen.MetricsBucketDurationDay,
	}
	_, err := tq.Metrics(context.Background(), input)
	require.NoError(t, err)
	mc.AssertExpectations(t)
}

func TestTorrentQuery_Metrics_WithStartEndTime(t *testing.T) {
	t.Parallel()

	mc := new(mockTorrentMetricsClient)
	tq := TorrentQuery{TorrentMetricsClient: mc}

	now := time.Now()

	mc.On("Request", mock.Anything, mock.MatchedBy(func(r torrentmetrics.Request) bool {
		return !r.StartTime.IsZero() && !r.EndTime.IsZero()
	})).Return([]torrentmetrics.Bucket{}, nil)

	input := gen.TorrentMetricsQueryInput{
		BucketDuration: gen.MetricsBucketDurationMinute,
		StartTime:      graphql.OmittableOf(&now),
		EndTime:        graphql.OmittableOf(&now),
	}
	_, err := tq.Metrics(context.Background(), input)
	require.NoError(t, err)
	mc.AssertExpectations(t)
}

func TestTorrentQuery_Metrics_WithTimezone(t *testing.T) {
	t.Parallel()

	mc := new(mockTorrentMetricsClient)
	tq := TorrentQuery{TorrentMetricsClient: mc}

	tz := "Asia/Shanghai"

	mc.On("Request", mock.Anything, mock.MatchedBy(func(r torrentmetrics.Request) bool {
		return r.Timezone == tz
	})).Return([]torrentmetrics.Bucket{}, nil)

	input := gen.TorrentMetricsQueryInput{
		BucketDuration: gen.MetricsBucketDurationDay,
		Timezone:       graphql.OmittableOf(&tz),
	}
	_, err := tq.Metrics(context.Background(), input)
	require.NoError(t, err)
	mc.AssertExpectations(t)
}

func TestTorrentQuery_Metrics_TimezoneDefaultsToUTC(t *testing.T) {
	t.Parallel()

	mc := new(mockTorrentMetricsClient)
	tq := TorrentQuery{TorrentMetricsClient: mc}

	mc.On("Request", mock.Anything, mock.MatchedBy(func(r torrentmetrics.Request) bool {
		return r.Timezone == "UTC"
	})).Return([]torrentmetrics.Bucket{}, nil)

	input := gen.TorrentMetricsQueryInput{
		BucketDuration: gen.MetricsBucketDurationDay,
	}
	_, err := tq.Metrics(context.Background(), input)
	require.NoError(t, err)
	mc.AssertExpectations(t)
}

func TestTorrentQuery_Metrics_Error(t *testing.T) {
	t.Parallel()

	mc := new(mockTorrentMetricsClient)
	tq := TorrentQuery{TorrentMetricsClient: mc}

	mc.On("Request", mock.Anything, mock.Anything).Return(nil, assert.AnError)

	input := gen.TorrentMetricsQueryInput{
		BucketDuration: gen.MetricsBucketDurationMinute,
	}
	_, err := tq.Metrics(context.Background(), input)
	require.Error(t, err)
	mc.AssertExpectations(t)
}
