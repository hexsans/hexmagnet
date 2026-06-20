package queuemetrics

import "time"

type Bucket struct {
	Queue           string
	Status          string
	CreatedAtBucket time.Time
	RanAtBucket     *time.Time
	Count           uint
	Latency         *time.Duration
}
