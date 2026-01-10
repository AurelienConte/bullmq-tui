package stats

import (
	"sync"
	"time"

	"github.com/AurelienConte/bullmq-tui/internal/redis"
)

type Collector struct {
	mu            sync.RWMutex
	buckets       map[string][]TimeBucket // queue -> time buckets
	windowMinutes int
	stats         map[string]*QueueStats
}

func NewCollector(windowMinutes int) *Collector {
	return &Collector{
		buckets:       make(map[string][]TimeBucket),
		windowMinutes: windowMinutes,
		stats:         make(map[string]*QueueStats),
	}
}

func (c *Collector) HandleEvent(event redis.JobEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()

	minute := time.Now().Truncate(time.Minute).Unix()
	queue := event.QueueName

	// Initialize if needed
	if c.buckets[queue] == nil {
		c.buckets[queue] = make([]TimeBucket, 0)
	}

	// Find or create bucket for current minute
	buckets := c.buckets[queue]
	var bucket *TimeBucket
	if len(buckets) > 0 && buckets[len(buckets)-1].Minute == minute {
		bucket = &buckets[len(buckets)-1]
	} else {
		c.buckets[queue] = append(buckets, TimeBucket{Minute: minute})
		bucket = &c.buckets[queue][len(c.buckets[queue])-1]
	}

	// Update counts
	switch event.Event {
	case "completed":
		bucket.Completed++
	case "failed":
		bucket.Failed++
	}

	// Prune old buckets
	c.pruneOldBuckets(queue)
}

func (c *Collector) pruneOldBuckets(queue string) {
	cutoff := time.Now().Add(-time.Duration(c.windowMinutes) * time.Minute).Unix()
	buckets := c.buckets[queue]

	// Find first bucket within window
	start := 0
	for i, b := range buckets {
		if b.Minute >= cutoff {
			start = i
			break
		}
	}

	c.buckets[queue] = buckets[start:]
}

func (c *Collector) GetStats(queue string) *QueueStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	buckets := c.buckets[queue]
	if len(buckets) == 0 {
		return &QueueStats{QueueName: queue}
	}

	// Calculate stats from buckets
	var totalCompleted, totalFailed int
	var peak int
	throughput := make([]int, len(buckets))

	for i, b := range buckets {
		count := b.Completed + b.Failed
		totalCompleted += b.Completed
		totalFailed += b.Failed
		throughput[i] = count
		if count > peak {
			peak = count
		}
	}

	elapsed := float64(len(buckets))
	if elapsed == 0 {
		elapsed = 1
	}

	var failureRate float64
	total := totalCompleted + totalFailed
	if total > 0 {
		failureRate = float64(totalFailed) / float64(total) * 100
	}

	return &QueueStats{
		QueueName:         queue,
		ProcessedTotal:    int64(totalCompleted),
		FailedTotal:       int64(totalFailed),
		JobsPerMinute:     float64(total) / elapsed,
		JobsPerMinutePeak: float64(peak),
		FailureRate:       failureRate,
		ThroughputHistory: throughput,
	}
}

func (c *Collector) GetAllStats() map[string]*QueueStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]*QueueStats)
	for queue := range c.buckets {
		result[queue] = c.GetStats(queue)
	}

	return result
}

func (c *Collector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.buckets = make(map[string][]TimeBucket)
	c.stats = make(map[string]*QueueStats)
}
