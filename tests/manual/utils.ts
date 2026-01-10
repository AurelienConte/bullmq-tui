/**
 * Promise-based sleep utility
 */
export function sleep(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms));
}

/**
 * Parse Redis URL to extract host and port
 */
export function parseRedisUrl(url: string): { host: string; port: number } {
  const match = url.match(/redis:\/\/([^:]+):(\d+)/);
  if (match) {
    return { host: match[1], port: parseInt(match[2]) };
  }
  return { host: 'localhost', port: 6379 };
}

/**
 * Format duration in milliseconds to human-readable string
 */
export function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${Math.floor(ms / 60000)}m ${Math.floor((ms % 60000) / 1000)}s`;
}

/**
 * Add terminal color codes to text
 */
export function colorize(text: string, color: 'green' | 'red' | 'yellow'): string {
  const colors = {
    green: '\x1b[32m',
    red: '\x1b[31m',
    yellow: '\x1b[33m',
    reset: '\x1b[0m'
  };
  return `${colors[color]}${text}${colors.reset}`;
}
