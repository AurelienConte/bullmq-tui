package stats

import "time"

type QueueStats struct {
	QueueName         string
	ProcessedTotal    int64
	FailedTotal       int64
	ProcessedToday    int64
	FailedToday       int64
	JobsPerMinute     float64
	JobsPerMinutePeak float64
	AvgWaitTime       time.Duration
	AvgProcessingTime time.Duration
	FailureRate       float64
	ThroughputHistory []int // Per-minute counts for sparkline
}

type TimeBucket struct {
	Minute    int64 // Unix timestamp truncated to minute
	Completed int
	Failed    int
}
