# BullMQ TUI Manual Testing Infrastructure

This directory contains a comprehensive manual testing setup for the BullMQ TUI. It includes Docker Compose for Redis, a TypeScript producer for dispatching jobs with various characteristics, and a TypeScript worker for processing them with different outcomes.

## Overview

The manual testing infrastructure allows you to test all features of the BullMQ TUI by simulating real-world job queue scenarios:

- **Producer**: Dispatches jobs at configurable rates with random intervals and optional delays
- **Worker**: Processes jobs with configurable success rates, progress updates, and realistic failures
- **Redis**: Containerized Redis instance for easy setup and cleanup

## Prerequisites

- **Docker** (with Docker Compose)
- **Node.js** 20+
- **make** (for running the TUI from the project root)

## Quick Start

1. **Start Redis**:
   ```bash
   npm run start:redis
   ```

2. **Install dependencies**:
   ```bash
   npm install
   ```

3. **Start the worker** (Terminal 1):
   ```bash
   npm run worker
   ```

4. **Start the producer** (Terminal 2):
   ```bash
   npm run producer
   ```

5. **Start the TUI** (Terminal 3):
   ```bash
   cd ../..
   make run-dev
   ```

You should now see jobs flowing through the system in real-time!

## Producer CLI Options

The producer script (`producer.ts`) supports the following options:

| Option | Short | Default | Description |
|--------|-------|---------|-------------|
| `--queue <name>` | `-q` | `test-queue` | Queue name to dispatch jobs to |
| `--rate <number>` | `-r` | `1` | Base rate in jobs per second |
| `--count <number>` | `-c` | `-1` | Total jobs to create (-1 for unlimited) |
| `--delay-chance <0-1>` | `-d` | `0.2` | Probability of adding delay to jobs (0.0 to 1.0) |
| `--max-delay <ms>` | `-m` | `30000` | Maximum delay for delayed jobs in milliseconds |
| `--redis <url>` | - | `redis://localhost:6379` | Redis connection URL |
| `--prefix <string>` | `-p` | `bull` | BullMQ key prefix (must match TUI config) |

### Job Types

The producer creates 4 different job types with varying probabilities:

- **simple-task** (40%): Basic task with minimal data, fast processing
- **slow-task** (20%): Takes 3-8 seconds with multiple operations
- **failing-task** (15%): Designed to fail with random error types
- **progress-task** (25%): Reports progress through multiple steps

### Dispatch Pattern

Jobs are dispatched with random intervals around the base rate (±50% variance) to simulate realistic, non-uniform traffic patterns.

## Worker CLI Options

The worker script (`worker.ts`) supports the following options:

| Option | Short | Default | Description |
|--------|-------|---------|-------------|
| `--queue <name>` | `-q` | `test-queue` | Queue name to process |
| `--concurrency <number>` | `-c` | `5` | Number of jobs to process concurrently |
| `--success-rate <0-1>` | `-s` | `0.7` | Success rate probability (0.0 to 1.0) |
| `--progress-chance <0-1>` | `-p` | `0.3` | Probability of jobs reporting progress |
| `--redis <url>` | - | `redis://localhost:6379` | Redis connection URL |
| `--prefix <string>` | - | `bull` | BullMQ key prefix (must match TUI config) |

### Processing Behavior

- **Successful jobs**: Process for 1-5 seconds and return success
- **Failed jobs**: Process for 1-3 seconds and throw realistic errors
- **Progress updates**: Report progress at regular intervals (0%, 25%, 50%, 75%, 100%)

## Testing Scenarios

### Basic Test

Single queue with default settings (1 job/sec, 70% success rate):

```bash
# Terminal 1
npm run worker

# Terminal 2
npm run producer

# Terminal 3
cd ../.. && make run-dev
```

### High Load Test

Test TUI performance with high job throughput:

```bash
# Terminal 1 - Worker with high concurrency
npm run worker -- -q load-test -c 20

# Terminal 2 - Producer at 10 jobs/sec
npm run producer -- -q load-test -r 10 -c 500

# Terminal 3
cd ../.. && make run-dev
```

### Failure Testing

Test TUI display of failed jobs with low success rate:

```bash
# Terminal 1 - Worker with 70% failure rate
npm run worker -- -q fail-test -s 0.3

# Terminal 2 - Producer
npm run producer -- -q fail-test -r 2

# Terminal 3
cd ../.. && make run-dev
```

### Delayed Jobs

Test delayed job queue with 80% of jobs delayed:

```bash
# Terminal 1
npm run worker -- -q delayed-test

# Terminal 2 - 80% delayed, up to 60 seconds
npm run producer -- -q delayed-test -r 1 -d 0.8 -m 60000

# Terminal 3
cd ../.. && make run-dev
```

### Progress Monitoring

Test progress updates in TUI:

```bash
# Terminal 1 - Worker with 80% progress jobs
npm run worker -- -q progress-test -p 0.8

# Terminal 2 - Slower rate to see progress clearly
npm run producer -- -q progress-test -r 0.5

# Terminal 3
cd ../.. && make run-dev
```

### Multi-Queue Testing

Test TUI queue switching with multiple queues:

```bash
# Terminal 1 - Multiple workers
npm run worker -- -q queue-a &
npm run worker -- -q queue-b -c 3 &

# Terminal 2 - Multiple producers
npm run producer -- -q queue-a -r 5 &
npm run producer -- -q queue-b -r 2 -d 0.5 &

# Terminal 3
cd ../.. && make run-dev
```

## NPM Scripts

Convenience scripts are available in `package.json`:

- `npm run producer` - Start the producer
- `npm run worker` - Start the worker
- `npm run start:redis` - Start Redis with Docker Compose
- `npm run stop:redis` - Stop Redis and remove container
- `npm run logs:redis` - View Redis logs
- `npm run test:basic` - Run basic test scenario (50 jobs at 2/sec)
- `npm run test:high-load` - Run high load test (500 jobs at 10/sec)
- `npm run test:failures` - Run worker with 70% failure rate
- `npm run test:delayed` - Run producer with 80% delayed jobs

## Troubleshooting

### Redis Connection Issues

**Problem**: `Could not connect to Redis after multiple attempts`

**Solution**:
- Ensure Redis is running: `docker compose ps`
- Check Redis logs: `npm run logs:redis`
- Verify port 6379 is not in use: `lsof -i :6379`

### Port Conflicts

**Problem**: Redis fails to start with port binding error

**Solution**:
- Stop any existing Redis instances: `docker compose down`
- Check for other services using port 6379: `lsof -i :6379`
- Kill the conflicting process or use a different port in `docker-compose.yml`

### Node Version Mismatch

**Problem**: TypeScript compilation errors or runtime issues

**Solution**:
- Check Node.js version: `node --version` (should be 20+)
- Update Node.js using nvm or your package manager
- Reinstall dependencies: `rm -rf node_modules && npm install`

### TUI Not Showing Updates

**Problem**: Jobs are being created but TUI doesn't show them

**Solution**:
- Verify BullMQ prefix matches: Both producer/worker use `--prefix bull` (default)
- Check TUI config: `~/.config/bullmq-tui/config.yaml` should have `prefix: "bull"`
- Ensure TUI is connected to the same Redis instance
- Try forcing refresh in TUI with `Ctrl+R`

### Worker Not Processing Jobs

**Problem**: Jobs stuck in "waiting" state

**Solution**:
- Ensure worker and producer use the same queue name
- Check worker logs for errors
- Verify worker is running: `ps aux | grep tsx`
- Restart worker: `Ctrl+C` then `npm run worker`

## Architecture Notes

### Job Flow

1. **Producer** adds jobs to Redis using BullMQ Queue API
2. **Redis** stores job data in hashes and lists/zsets for each state
3. **Worker** picks up jobs and processes them
4. **TUI** monitors Redis keys and pub/sub events to display real-time updates

### BullMQ Prefix Alignment

The TUI uses a configurable prefix for BullMQ keys (default: `"bull"`). Both the producer and worker must use the same prefix to ensure jobs are visible in the TUI.

Redis key structure:
```
bull:<queue>:meta       # Queue metadata
bull:<queue>:wait       # Waiting jobs
bull:<queue>:active     # Active jobs
bull:<queue>:delayed    # Delayed jobs (ZSet)
bull:<queue>:completed  # Completed jobs (ZSet)
bull:<queue>:failed     # Failed jobs (ZSet)
bull:<queue>:<jobID>    # Job data
bull:<queue>:events     # Pub/sub events
```

### Job State Lifecycle

1. **waiting** → Job added to queue, waiting for worker
2. **delayed** → Job scheduled for future execution
3. **active** → Worker picked up job, currently processing
4. **completed** → Job finished successfully
5. **failed** → Job threw an error during processing

The TUI displays counts for each state and allows navigation between them.

## What This Tests

This infrastructure enables comprehensive testing of TUI features:

- **Queue Discovery**: Multiple queues appear automatically
- **Job State Display**: All 5 states (waiting, active, delayed, completed, failed)
- **Real-time Updates**: Counts update as jobs progress
- **Stats Collection**: Jobs/min, failure rate, throughput graphs
- **Job Details**: View JSON data, timestamps, stack traces
- **Progress Monitoring**: 0-100% progress updates
- **Event Streaming**: Redis pub/sub integration
- **Multiple Queues**: Navigate between different queues
- **High Load**: Performance with 10+ jobs/second
- **Error Handling**: Display of failed jobs with error messages

## Stopping Services

To clean up all services:

```bash
# Stop Redis
npm run stop:redis

# Kill producer/worker (Ctrl+C in their terminals or)
pkill -f "tsx producer.ts"
pkill -f "tsx worker.ts"

# Exit TUI (press 'q' or Ctrl+C)
```

## Development

### Modifying Job Types

Edit the `JOB_FACTORIES` array in `producer.ts` to add new job types or change probabilities.

### Adding Failure Reasons

Edit the `FAILURE_REASONS` array in `worker.ts` to add new realistic error messages.

### Changing Processing Logic

Modify the `processJob` function in `worker.ts` to customize how jobs are processed.

## License

This testing infrastructure is part of the BullMQ TUI project and shares the same license.
