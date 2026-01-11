import { Queue } from 'bullmq';
import { Command } from 'commander';
import Redis from 'ioredis';
import { sleep, parseRedisUrl, colorize } from './utils.js';

/**
 * Job factory interface
 */
interface JobFactory {
  type: string;
  probability: number;
  generate(): { name: string; data: any };
}

/**
 * Job factories for different job types
 */
const JOB_FACTORIES: JobFactory[] = [
  {
    type: 'simple',
    probability: 0.4,
    generate: () => ({
      name: 'simple-task',
      data: {
        id: Math.random().toString(36).substring(7),
        timestamp: Date.now(),
        message: 'Simple task with minimal data'
      }
    })
  },
  {
    type: 'slow',
    probability: 0.2,
    generate: () => ({
      name: 'slow-task',
      data: {
        id: Math.random().toString(36).substring(7),
        duration: Math.floor(Math.random() * 5000) + 3000,
        operations: ['step1', 'step2', 'step3']
      }
    })
  },
  {
    type: 'failing',
    probability: 0.15,
    generate: () => ({
      name: 'failing-task',
      data: {
        id: Math.random().toString(36).substring(7),
        shouldFail: true,
        errorType: ['validation', 'timeout', 'network'][Math.floor(Math.random() * 3)]
      }
    })
  },
  {
    type: 'with-progress',
    probability: 0.25,
    generate: () => ({
      name: 'progress-task',
      data: {
        id: Math.random().toString(36).substring(7),
        steps: Math.floor(Math.random() * 5) + 3,
        payload: Buffer.from(Math.random().toString()).toString('base64')
      }
    })
  }
];

/**
 * Select a job factory based on probabilities
 */
function selectJobFactory(): JobFactory {
  const rand = Math.random();
  let cumulative = 0;

  for (const factory of JOB_FACTORIES) {
    cumulative += factory.probability;
    if (rand <= cumulative) {
      return factory;
    }
  }

  return JOB_FACTORIES[0];
}

/**
 * Calculate next delay with random jitter
 */
function calculateNextDelay(baseRatePerSec: number): number {
  const baseDelay = 1000 / baseRatePerSec;
  const jitter = (Math.random() - 0.5) * baseDelay;
  return Math.max(10, baseDelay + jitter);
}

/**
 * Determine if job should be delayed
 */
function shouldDelayJob(delayChance: number, maxDelayMs: number): number | undefined {
  if (Math.random() < delayChance) {
    const minDelay = 5000;
    const range = maxDelayMs - minDelay;
    return Math.floor(Math.random() * range) + minDelay;
  }
  return undefined;
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
 * Producer options
 */
interface ProducerOptions {
  queue: string;
  rate: number;
  count: number;
  delayChance: number;
  maxDelay: number;
  redis: string;
  prefix: string;
}

/**
 * Run the producer
 */
async function runProducer(options: ProducerOptions) {
  try {
    await waitForRedis(options.redis);
  } catch (err) {
    console.error(colorize(`✗ ${(err as Error).message}`, 'red'));
    process.exit(1);
  }

  const { host, port } = parseRedisUrl(options.redis);
  const queue = new Queue(options.queue, {
    connection: {
      host,
      port,
    },
    prefix: options.prefix,
  });

  let jobCount = 0;
  let running = true;

  process.on('SIGINT', async () => {
    console.log(colorize('\n🛑 Shutting down producer...', 'yellow'));
    running = false;
    await queue.close();
    console.log(colorize('✓ Shutdown complete', 'green'));
    process.exit(0);
  });

  process.on('SIGTERM', async () => {
    console.log(colorize('\n🛑 Shutting down producer...', 'yellow'));
    running = false;
    await queue.close();
    console.log(colorize('✓ Shutdown complete', 'green'));
    process.exit(0);
  });

  console.log(colorize('Producer started:', 'green'));
  console.log(`  Queue: ${options.queue}`);
  console.log(`  Rate: ${options.rate} jobs/sec`);
  console.log(`  Delay chance: ${options.delayChance * 100}%`);
  console.log(`  Total: ${options.count === -1 ? 'unlimited' : options.count}`);
  console.log('');

  while (running && (options.count === -1 || jobCount < options.count)) {
    const factory = selectJobFactory();
    const { name, data } = factory.generate();

    const delay = shouldDelayJob(options.delayChance, options.maxDelay);
    const jobOptions: any = {};

    if (delay) {
      jobOptions.delay = delay;
    }

    try {
      const job = await queue.add(name, data, jobOptions);
      jobCount++;

      const delayStr = delay ? colorize(` (delayed ${Math.round(delay/1000)}s)`, 'yellow') : '';
      console.log(`[${jobCount}] Added ${colorize(name, 'green')} #${job.id}${delayStr}`);
    } catch (err) {
      console.error(colorize(`Failed to add job: ${(err as Error).message}`, 'red'));
    }

    const nextDelay = calculateNextDelay(options.rate);
    await sleep(nextDelay);
  }

  console.log(colorize(`\n✓ Producer finished: ${jobCount} jobs dispatched`, 'green'));
  await queue.close();
  process.exit(0);
}

const program = new Command();

program
  .name('producer')
  .description('BullMQ job producer for testing')
  .option('-q, --queue <name>', 'Queue name', 'test-queue')
  .option('-r, --rate <number>', 'Jobs per second', parseFloat, 1)
  .option('-c, --count <number>', 'Total jobs to create (-1 for unlimited)', parseInt, -1)
  .option('-d, --delay-chance <0-1>', 'Probability of delayed jobs', parseFloat, 0.2)
  .option('-m, --max-delay <ms>', 'Maximum delay for delayed jobs', parseInt, 30000)
  .option('--redis <url>', 'Redis connection URL', 'redis://localhost:6379')
  .option('-p, --prefix <string>', 'BullMQ prefix', 'bull')
  .action(runProducer);

program.parse();
