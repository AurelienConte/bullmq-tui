import { Worker, Job } from 'bullmq';
import { Command } from 'commander';
import Redis from 'ioredis';
import { sleep, parseRedisUrl, colorize } from './utils.js';

/**
 * Pool of realistic failure reasons
 */
const FAILURE_REASONS = [
  'Connection timeout to external API',
  'Invalid data format: missing required field',
  'Database constraint violation',
  'Rate limit exceeded on third-party service',
  'File not found: /tmp/missing-file.txt',
  'Memory allocation failed',
  'Network unreachable: host down',
  'Authentication failed: invalid credentials'
];

/**
 * Select a random failure reason
 */
function selectRandomFailure(): string {
  return FAILURE_REASONS[Math.floor(Math.random() * FAILURE_REASONS.length)];
}

/**
 * Get duration for job based on type
 */
function getDurationForJob(job: Job): number {
  if (job.data.duration) {
    return job.data.duration;
  }

  switch (job.name) {
    case 'slow-task':
      return Math.floor(Math.random() * 5000) + 3000;
    case 'simple-task':
      return Math.floor(Math.random() * 1000) + 500;
    default:
      return Math.floor(Math.random() * 2000) + 1000;
  }
}

/**
 * Simulate work with a delay
 */
async function simulateWork(minMs: number, maxMs: number): Promise<void> {
  const duration = Math.floor(Math.random() * (maxMs - minMs)) + minMs;
  await sleep(duration);
}

/**
 * Process job with progress updates
 */
async function processWithProgress(job: Job): Promise<any> {
  const steps = job.data.steps || 5;
  const totalDuration = Math.floor(Math.random() * 4000) + 2000;
  const stepDuration = totalDuration / steps;

  for (let i = 0; i <= steps; i++) {
    const progress = Math.round((i / steps) * 100);

    await job.updateProgress({
      percent: progress,
      step: i,
      totalSteps: steps,
      message: `Processing step ${i}/${steps}`
    });

    console.log(`[${job.id}] ⟳ ${progress}% - Processing step ${i}/${steps}`);

    if (i < steps) {
      await sleep(stepDuration);
    }
  }

  return {
    success: true,
    steps,
    completedAt: new Date().toISOString()
  };
}

/**
 * Process a job
 */
async function processJob(
  job: Job,
  successRate: number,
  progressChance: number
): Promise<any> {
  const willSucceed = Math.random() < successRate;
  const hasProgress = Math.random() < progressChance;

  console.log(`[${job.id}] Processing ${colorize(job.name, 'yellow')} (${willSucceed ? colorize('will succeed', 'green') : colorize('will fail', 'red')})`);

  if (job.name === 'failing-task' || job.data.shouldFail) {
    await simulateWork(1000, 3000);
    throw new Error(selectRandomFailure());
  }

  if (!willSucceed) {
    const duration = getDurationForJob(job);
    await simulateWork(duration * 0.3, duration * 0.7);
    throw new Error(selectRandomFailure());
  }

  if (hasProgress || job.name === 'progress-task') {
    return await processWithProgress(job);
  }

  const duration = getDurationForJob(job);
  await simulateWork(duration, duration);

  return {
    success: true,
    processedAt: new Date().toISOString(),
    jobType: job.name,
    duration
  };
}

/**
 * Wait for Redis to be available
 */
async function waitForRedis(redisUrl: string, maxRetries: number = 10): Promise<void> {
  const redis = new Redis(redisUrl);

  for (let i = 0; i < maxRetries; i++) {
    try {
      await redis.ping();
      console.log(colorize('✓ Connected to Redis', 'green'));
      await redis.quit();
      return;
    } catch (err) {
      console.log(colorize(`Waiting for Redis... (${i + 1}/${maxRetries})`, 'yellow'));
      await sleep(2000);
    }
  }

  await redis.quit();
  throw new Error('Could not connect to Redis after multiple attempts');
}

/**
 * Worker options
 */
interface WorkerOptions {
  queue: string;
  concurrency: number;
  successRate: number;
  progressChance: number;
  redis: string;
  prefix: string;
}

/**
 * Setup worker with event listeners
 */
function setupWorker(options: WorkerOptions): Worker {
  const { host, port } = parseRedisUrl(options.redis);

  const worker = new Worker(
    options.queue,
    async (job) => processJob(job, options.successRate, options.progressChance),
    {
      connection: {
        host,
        port,
      },
      prefix: options.prefix,
      concurrency: options.concurrency,
    }
  );

  worker.on('active', (job) => {
    console.log(`[${job.id}] ▶ Active: ${colorize(job.name, 'green')}`);
  });

  worker.on('completed', (job) => {
    const duration = Date.now() - job.timestamp;
    console.log(`[${job.id}] ${colorize('✓', 'green')} Completed in ${duration}ms`);
  });

  worker.on('failed', (job, err) => {
    if (job) {
      console.log(`[${job.id}] ${colorize('✗', 'red')} Failed: ${err.message}`);
    } else {
      console.log(`${colorize('✗', 'red')} Job failed: ${err.message}`);
    }
  });

  worker.on('progress', (job, progress: any) => {
    if (typeof progress === 'object' && progress.percent !== undefined) {
      console.log(`[${job.id}] ⟳ ${progress.percent}% - ${progress.message}`);
    }
  });

  worker.on('error', (err) => {
    console.error(colorize(`Worker error: ${err.message}`, 'red'));
  });

  return worker;
}

/**
 * Run the worker
 */
async function runWorker(options: WorkerOptions) {
  try {
    await waitForRedis(options.redis);
  } catch (err) {
    console.error(colorize(`✗ ${(err as Error).message}`, 'red'));
    process.exit(1);
  }

  console.log(colorize('Worker started:', 'green'));
  console.log(`  Queue: ${options.queue}`);
  console.log(`  Concurrency: ${options.concurrency}`);
  console.log(`  Success rate: ${options.successRate * 100}%`);
  console.log(`  Progress chance: ${options.progressChance * 100}%`);
  console.log('');

  const worker = setupWorker(options);

  process.on('SIGINT', async () => {
    console.log(colorize('\n🛑 Shutting down worker...', 'yellow'));
    await worker.close();
    console.log(colorize('✓ Shutdown complete', 'green'));
    process.exit(0);
  });

  process.on('SIGTERM', async () => {
    console.log(colorize('\n🛑 Shutting down worker...', 'yellow'));
    await worker.close();
    console.log(colorize('✓ Shutdown complete', 'green'));
    process.exit(0);
  });

  await new Promise(() => {});
}

const program = new Command();

program
  .name('worker')
  .description('BullMQ job worker for testing')
  .option('-q, --queue <name>', 'Queue name to process', 'test-queue')
  .option('-c, --concurrency <number>', 'Concurrent jobs', parseInt, 5)
  .option('-s, --success-rate <0-1>', 'Success rate probability', parseFloat, 0.7)
  .option('-p, --progress-chance <0-1>', 'Jobs with progress', parseFloat, 0.3)
  .option('--redis <url>', 'Redis connection URL', 'redis://localhost:6379')
  .option('--prefix <string>', 'BullMQ prefix', 'bull')
  .action(runWorker);

program.parse();
