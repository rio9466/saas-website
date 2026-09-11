package redis

import (
	"context"
	"fmt"

	goredis "github.com/redis/go-redis/v9"
	"github.com/rio9466/easy-admin/server/internal/config"
)

func init() {
	// Prevent go-redis from writing dial/pool diagnostics to the process log.
	goredis.SetLogger(silentRedisLogger{})
}

type silentRedisLogger struct{}

func (silentRedisLogger) Printf(context.Context, string, ...interface{}) {}

// Client wraps a go-redis client with startup verification and cleanup.
type Client struct {
	rdb *goredis.Client
}

// Open creates a Redis client and verifies connectivity within the startup timeout.
func Open(ctx context.Context, cfg config.RedisConfig) (*Client, error) {
	rdb := goredis.NewClient(&goredis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})

	client := &Client{rdb: rdb}

	startupCtx, cancel := context.WithTimeout(ctx, cfg.StartupTimeout)
	defer cancel()

	if err := client.Ping(startupCtx); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis startup ping: %w", err)
	}

	return client, nil
}

// Raw returns the underlying go-redis client for session and cache use.
func (c *Client) Raw() *goredis.Client {
	if c == nil {
		return nil
	}
	return c.rdb
}

// Ping verifies Redis connectivity using the provided context deadline.
func (c *Client) Ping(ctx context.Context) error {
	if c == nil || c.rdb == nil {
		return fmt.Errorf("redis is not initialized")
	}
	if err := c.rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("ping redis: %w", err)
	}
	return nil
}

// Close releases the underlying Redis client.
func (c *Client) Close() error {
	if c == nil || c.rdb == nil {
		return nil
	}
	if err := c.rdb.Close(); err != nil {
		return fmt.Errorf("close redis: %w", err)
	}
	return nil
}
