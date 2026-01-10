package redis

import "time"

type JobState string

const (
	JobStateWaiting   JobState = "waiting"
	JobStateActive    JobState = "active"
	JobStateDelayed   JobState = "delayed"
	JobStateCompleted JobState = "completed"
	JobStateFailed    JobState = "failed"
	JobStatePaused    JobState = "paused"
)

type Queue struct {
	Name        string
	Prefix      string
	IsPaused    bool
	Counts      QueueCounts
	FailureRate float64
	HasFailures bool
}

type QueueCounts struct {
	Waiting   int64
	Active    int64
	Delayed   int64
	Completed int64
	Failed    int64
	Paused    int64
}

type Job struct {
	ID           string
	Name         string
	QueueName    string
	State        JobState
	Data         string      // JSON string
	Opts         string      // JSON string
	Progress     interface{} // Can be number or object
	Delay        int64
	Priority     int
	Attempts     int
	AttemptsMade int
	Timestamp    time.Time  // When job was created
	ProcessedOn  *time.Time // When processing started
	FinishedOn   *time.Time // When job completed/failed
	ReturnValue  string     // JSON string, result on completion
	FailedReason string
	Stacktrace   []string
}

type JobEvent struct {
	Event     string    // "completed", "failed", "active", etc.
	JobID     string
	QueueName string
	Timestamp time.Time
	Data      map[string]interface{}
}
