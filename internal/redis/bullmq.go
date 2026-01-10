package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type BullMQClient struct {
	client redis.Cmdable
	prefix string
}

func NewBullMQClient(client redis.Cmdable, prefix string) *BullMQClient {
	if prefix == "" {
		prefix = "bull"
	}
	return &BullMQClient{client: client, prefix: prefix}
}

// key builds a BullMQ key with the given queue name and suffix
func (b *BullMQClient) key(queueName, suffix string) string {
	return fmt.Sprintf("%s:%s:%s", b.prefix, queueName, suffix)
}

// DiscoverQueues finds all BullMQ queues by scanning for meta keys
func (b *BullMQClient) DiscoverQueues(ctx context.Context) ([]string, error) {
	pattern := fmt.Sprintf("%s:*:meta", b.prefix)

	var queues []string
	queueMap := make(map[string]bool)

	iter := b.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		// Extract queue name: bull:<queue>:meta -> <queue>
		parts := strings.Split(key, ":")
		if len(parts) >= 3 {
			queueName := parts[1]
			if !queueMap[queueName] {
				queueMap[queueName] = true
				queues = append(queues, queueName)
			}
		}
	}

	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan for queues: %w", err)
	}

	return queues, nil
}

// GetQueueCounts retrieves job counts for all states
func (b *BullMQClient) GetQueueCounts(ctx context.Context, queueName string) (*QueueCounts, error) {
	pipe := b.client.Pipeline()

	// Queue all count commands
	waitCmd := pipe.LLen(ctx, b.key(queueName, "wait"))
	activeCmd := pipe.LLen(ctx, b.key(queueName, "active"))
	delayedCmd := pipe.ZCard(ctx, b.key(queueName, "delayed"))
	completedCmd := pipe.ZCard(ctx, b.key(queueName, "completed"))
	failedCmd := pipe.ZCard(ctx, b.key(queueName, "failed"))
	pausedCmd := pipe.LLen(ctx, b.key(queueName, "paused"))

	// Execute pipeline
	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to get queue counts: %w", err)
	}

	return &QueueCounts{
		Waiting:   waitCmd.Val(),
		Active:    activeCmd.Val(),
		Delayed:   delayedCmd.Val(),
		Completed: completedCmd.Val(),
		Failed:    failedCmd.Val(),
		Paused:    pausedCmd.Val(),
	}, nil
}

// IsQueuePaused checks if a queue is paused
func (b *BullMQClient) IsQueuePaused(ctx context.Context, queueName string) (bool, error) {
	result := b.client.HExists(ctx, b.key(queueName, "meta"), "paused")
	if err := result.Err(); err != nil && err != redis.Nil {
		return false, err
	}
	return result.Val(), nil
}

// PauseQueue pauses a queue
func (b *BullMQClient) PauseQueue(ctx context.Context, queueName string) error {
	return b.client.HSet(ctx, b.key(queueName, "meta"), "paused", 1).Err()
}

// ResumeQueue resumes a paused queue
func (b *BullMQClient) ResumeQueue(ctx context.Context, queueName string) error {
	return b.client.HDel(ctx, b.key(queueName, "meta"), "paused").Err()
}

// GetJobs retrieves jobs for a specific state
func (b *BullMQClient) GetJobs(ctx context.Context, queueName string, state JobState, start, end int64) ([]Job, error) {
	var jobIDs []string
	var err error

	// Get job IDs based on state
	switch state {
	case JobStateWaiting:
		jobIDs, err = b.client.LRange(ctx, b.key(queueName, "wait"), start, end).Result()
	case JobStateActive:
		jobIDs, err = b.client.LRange(ctx, b.key(queueName, "active"), start, end).Result()
	case JobStatePaused:
		jobIDs, err = b.client.LRange(ctx, b.key(queueName, "paused"), start, end).Result()
	case JobStateDelayed, JobStateCompleted, JobStateFailed:
		// These are sorted sets
		jobIDs, err = b.client.ZRange(ctx, b.key(queueName, string(state)), start, end).Result()
	default:
		return nil, fmt.Errorf("unsupported job state: %s", state)
	}

	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to get job IDs: %w", err)
	}

	if len(jobIDs) == 0 {
		return []Job{}, nil
	}

	// Fetch job data
	jobs := make([]Job, 0, len(jobIDs))
	for _, jobID := range jobIDs {
		job, err := b.GetJob(ctx, queueName, jobID)
		if err != nil {
			// Skip jobs that no longer exist
			continue
		}
		job.State = state
		jobs = append(jobs, *job)
	}

	return jobs, nil
}

// GetJob retrieves a single job by ID
func (b *BullMQClient) GetJob(ctx context.Context, queueName string, jobID string) (*Job, error) {
	key := b.key(queueName, jobID)

	result := b.client.HGetAll(ctx, key)
	if err := result.Err(); err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	data := result.Val()
	if len(data) == 0 {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}

	job := &Job{
		ID:        jobID,
		QueueName: queueName,
	}

	// Parse job fields
	if v, ok := data["name"]; ok {
		job.Name = v
	}
	if v, ok := data["data"]; ok {
		job.Data = v
	}
	if v, ok := data["opts"]; ok {
		job.Opts = v
	}
	if v, ok := data["progress"]; ok {
		// Try to parse as JSON
		var progress interface{}
		if err := json.Unmarshal([]byte(v), &progress); err == nil {
			job.Progress = progress
		}
	}
	if v, ok := data["delay"]; ok {
		if d, err := strconv.ParseInt(v, 10, 64); err == nil {
			job.Delay = d
		}
	}
	if v, ok := data["priority"]; ok {
		if p, err := strconv.Atoi(v); err == nil {
			job.Priority = p
		}
	}
	if v, ok := data["attempts"]; ok {
		if a, err := strconv.Atoi(v); err == nil {
			job.Attempts = a
		}
	}
	if v, ok := data["attemptsMade"]; ok {
		if a, err := strconv.Atoi(v); err == nil {
			job.AttemptsMade = a
		}
	}
	if v, ok := data["timestamp"]; ok {
		if ts, err := strconv.ParseInt(v, 10, 64); err == nil {
			job.Timestamp = time.Unix(ts/1000, 0)
		}
	}
	if v, ok := data["processedOn"]; ok {
		if ts, err := strconv.ParseInt(v, 10, 64); err == nil {
			t := time.Unix(ts/1000, 0)
			job.ProcessedOn = &t
		}
	}
	if v, ok := data["finishedOn"]; ok {
		if ts, err := strconv.ParseInt(v, 10, 64); err == nil {
			t := time.Unix(ts/1000, 0)
			job.FinishedOn = &t
		}
	}
	if v, ok := data["returnvalue"]; ok {
		job.ReturnValue = v
	}
	if v, ok := data["failedReason"]; ok {
		job.FailedReason = v
	}
	if v, ok := data["stacktrace"]; ok {
		// Parse stacktrace as JSON array
		var st []string
		if err := json.Unmarshal([]byte(v), &st); err == nil {
			job.Stacktrace = st
		}
	}

	return job, nil
}

// RetryJob moves a job from failed to waiting
func (b *BullMQClient) RetryJob(ctx context.Context, queueName string, jobID string) error {
	pipe := b.client.Pipeline()

	// Remove from failed set
	pipe.ZRem(ctx, b.key(queueName, "failed"), jobID)

	// Add to wait list
	pipe.RPush(ctx, b.key(queueName, "wait"), jobID)

	// Update job hash: reset attemptsMade and clear failedReason
	pipe.HSet(ctx, b.key(queueName, jobID), "attemptsMade", 0)
	pipe.HDel(ctx, b.key(queueName, jobID), "failedReason", "stacktrace")

	_, err := pipe.Exec(ctx)
	return err
}

// RetryAllFailed moves all failed jobs back to waiting
func (b *BullMQClient) RetryAllFailed(ctx context.Context, queueName string) (int64, error) {
	// Get all failed job IDs
	jobIDs, err := b.client.ZRange(ctx, b.key(queueName, "failed"), 0, -1).Result()
	if err != nil {
		return 0, err
	}

	count := int64(0)
	for _, jobID := range jobIDs {
		if err := b.RetryJob(ctx, queueName, jobID); err != nil {
			continue
		}
		count++
	}

	return count, nil
}

// DeleteJob removes a job completely
func (b *BullMQClient) DeleteJob(ctx context.Context, queueName string, jobID string, state JobState) error {
	pipe := b.client.Pipeline()

	// Remove from state list/set
	switch state {
	case JobStateWaiting:
		pipe.LRem(ctx, b.key(queueName, "wait"), 0, jobID)
	case JobStateActive:
		pipe.LRem(ctx, b.key(queueName, "active"), 0, jobID)
	case JobStatePaused:
		pipe.LRem(ctx, b.key(queueName, "paused"), 0, jobID)
	case JobStateDelayed, JobStateCompleted, JobStateFailed:
		pipe.ZRem(ctx, b.key(queueName, string(state)), jobID)
	}

	// Delete job hash
	pipe.Del(ctx, b.key(queueName, jobID))

	_, err := pipe.Exec(ctx)
	return err
}

// DrainQueue removes all jobs in a given state
func (b *BullMQClient) DrainQueue(ctx context.Context, queueName string, state JobState) (int64, error) {
	// Get all job IDs
	jobs, err := b.GetJobs(ctx, queueName, state, 0, -1)
	if err != nil {
		return 0, err
	}

	count := int64(0)
	for _, job := range jobs {
		if err := b.DeleteJob(ctx, queueName, job.ID, state); err != nil {
			continue
		}
		count++
	}

	return count, nil
}

// AddJob adds a new job to a queue
func (b *BullMQClient) AddJob(ctx context.Context, queueName string, jobName string, jobData map[string]interface{}, opts map[string]interface{}) (string, error) {
	// 1. Increment job ID counter
	jobID, err := b.client.Incr(ctx, b.key(queueName, "id")).Result()
	if err != nil {
		return "", fmt.Errorf("failed to generate job ID: %w", err)
	}
	jobIDStr := strconv.FormatInt(jobID, 10)

	// 2. Marshal data and opts to JSON
	timestamp := time.Now().UnixMilli()
	dataJSON, _ := json.Marshal(jobData)
	optsJSON, _ := json.Marshal(opts)
	if optsJSON == nil {
		optsJSON = []byte("{}")
	}

	// 3. Create job hash + add to waiting queue + ensure meta exists (pipeline)
	pipe := b.client.Pipeline()
	jobKey := b.key(queueName, jobIDStr)
	pipe.HSet(ctx, jobKey, map[string]interface{}{
		"name":         jobName,
		"data":         string(dataJSON),
		"opts":         string(optsJSON),
		"timestamp":    timestamp,
		"attempts":     3,
		"attemptsMade": 0,
	})
	pipe.LPush(ctx, b.key(queueName, "wait"), jobIDStr)
	pipe.HSetNX(ctx, b.key(queueName, "meta"), "created", timestamp)

	if _, err := pipe.Exec(ctx); err != nil {
		return "", fmt.Errorf("failed to create job: %w", err)
	}

	return jobIDStr, nil
}
