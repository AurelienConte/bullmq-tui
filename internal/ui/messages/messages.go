package messages

import (
	"time"

	"github.com/AurelienConte/bullmq-tui/internal/redis"
	"github.com/AurelienConte/bullmq-tui/internal/stats"
)

// Data messages
type QueuesUpdated []redis.Queue
type JobsUpdated []redis.Job
type StatsUpdated *stats.QueueStats
type JobEvent redis.JobEvent

// Action messages
type QueueSelected string
type StateSelected redis.JobState
type JobSelected string

// Result messages
type JobRetried struct{ ID string }
type JobDeleted struct{ ID string }
type QueueDrained struct{ Count int64 }
type QueuePaused struct{ Name string }
type QueueResumed struct{ Name string }

// Error
type Error error

// Tick for periodic refresh
type Tick time.Time
