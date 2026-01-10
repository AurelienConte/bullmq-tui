package redis

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/AurelienConte/bullmq-tui/internal/config"
	"github.com/redis/go-redis/v9"
)

// NewClient creates a new Redis client from a connection config
func NewClient(ctx context.Context, conn *config.Connection) (*redis.Client, error) {
	opts := &redis.Options{
		Addr:     fmt.Sprintf("%s:%d", conn.Host, conn.Port),
		Password: conn.Password,
		DB:       conn.DB,
	}

	// TLS configuration
	if conn.TLS {
		opts.TLSConfig = &tls.Config{
			InsecureSkipVerify: conn.TLSSkipVerify,
		}
	}

	client := redis.NewClient(opts)

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return client, nil
}

// NewClusterClient creates a new Redis cluster client
func NewClusterClient(ctx context.Context, conn *config.Connection) (*redis.ClusterClient, error) {
	if conn.Cluster == nil || !conn.Cluster.Enabled {
		return nil, fmt.Errorf("cluster not configured")
	}

	opts := &redis.ClusterOptions{
		Addrs:    conn.Cluster.Addresses,
		Password: conn.Password,
	}

	// TLS configuration
	if conn.TLS {
		opts.TLSConfig = &tls.Config{
			InsecureSkipVerify: conn.TLSSkipVerify,
		}
	}

	client := redis.NewClusterClient(opts)

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis cluster: %w", err)
	}

	return client, nil
}
