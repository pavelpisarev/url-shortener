package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const clickQueueKey = "clicks:queue"

type Client struct {
	rdb *goredis.Client
}

func New(ctx context.Context, url string) (*Client, error) {
	opts, err := goredis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}

	rdb := goredis.NewClient(opts)
	c := &Client{rdb: rdb}
	if err := c.Ping(ctx); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return c, nil
}

func (c *Client) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

func (c *Client) Close() error {
	return c.rdb.Close()
}

func (c *Client) Get(ctx context.Context, code string) (string, bool, error) {
	v, err := c.rdb.Get(ctx, cacheKey(code)).Result()
	if errors.Is(err, goredis.Nil) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return v, true, nil
}

func (c *Client) Set(ctx context.Context, code, url string, ttl time.Duration) error {
	return c.rdb.Set(ctx, cacheKey(code), url, ttl).Err()
}

func (c *Client) EnqueueClick(ctx context.Context, code string) error {
	return c.rdb.LPush(ctx, clickQueueKey, code).Err()
}

func (c *Client) PopClicks(ctx context.Context, max int) ([]string, error) {
	if max < 1 {
		max = 1
	}

	res, err := c.rdb.BRPop(ctx, 2*time.Second, clickQueueKey).Result()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return nil, nil
		}
		return nil, err
	}
	if len(res) < 2 {
		return nil, nil
	}

	codes := []string{res[1]}
	for len(codes) < max {
		v, err := c.rdb.LPop(ctx, clickQueueKey).Result()
		if errors.Is(err, goredis.Nil) {
			break
		}
		if err != nil {
			return codes, err
		}
		codes = append(codes, v)
	}
	return codes, nil
}

func cacheKey(code string) string {
	return "link:" + code
}
