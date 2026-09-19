package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"goshort/model"
)

const (
	defaultRedisAddr = "localhost:6379"
	defaultRedisDB   = 0

	defaultURLCacheTTL = 1 * time.Hour
)

var ErrCacheMiss = errors.New("cache miss")

type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisCache() (*RedisCache, error) {
	addr := os.Getenv("GOSHORT_REDIS_ADDR")
	if addr == "" {
		addr = defaultRedisAddr
	}

	password := os.Getenv("GOSHORT_REDIS_PASSWORD")

	db := defaultRedisDB

	if rawDB := os.Getenv("GOSHORT_REDIS_DB"); rawDB != "" {
		parsedDB, err := strconv.Atoi(rawDB)
		if err != nil || parsedDB < 0 {
			return nil, fmt.Errorf("invalid GOSHORT_REDIS_DB")
		}

		db = parsedDB
	}

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,

		PoolSize:     10,
		MinIdleConns: 2,
		MaxIdleConns: 5,

		DialTimeout:  3 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})

	return &RedisCache{
		client: client,
		ttl:    defaultURLCacheTTL,
	}, nil
}

func (c *RedisCache) Close() error {
	if c == nil || c.client == nil {
		return nil
	}

	return c.client.Close()
}

func (c *RedisCache) Ping(ctx context.Context) error {
	if c == nil || c.client == nil {
		return errors.New("redis cache is not initialized")
	}

	return c.client.Ping(ctx).Err()
}

func (c *RedisCache) GetURL(
	ctx context.Context,
	shortCode string,
) (model.URL, error) {
	if c == nil || c.client == nil {
		return model.URL{}, ErrCacheMiss
	}

	key := urlCacheKey(shortCode)

	value, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return model.URL{}, ErrCacheMiss
		}

		return model.URL{}, fmt.Errorf("redis get: %w", err)
	}

	var cachedURL model.URL

	if err := json.Unmarshal(value, &cachedURL); err != nil {
		_ = c.client.Del(ctx, key).Err()

		return model.URL{}, fmt.Errorf(
			"decode cached URL: %w",
			err,
		)
	}

	return cachedURL, nil
}

func (c *RedisCache) SetURL(
	ctx context.Context,
	url model.URL,
) error {
	if c == nil || c.client == nil {
		return errors.New("redis cache is not initialized")
	}

	value, err := json.Marshal(url)
	if err != nil {
		return fmt.Errorf("encode URL for cache: %w", err)
	}

	key := urlCacheKey(url.ShortCode)

	if err := c.client.Set(
		ctx,
		key,
		value,
		c.cacheTTL(url),
	).Err(); err != nil {
		return fmt.Errorf("redis set: %w", err)
	}

	return nil
}

func (c *RedisCache) DeleteURL(
	ctx context.Context,
	shortCode string,
) error {
	if c == nil || c.client == nil {
		return nil
	}

	if err := c.client.Del(
		ctx,
		urlCacheKey(shortCode),
	).Err(); err != nil {
		return fmt.Errorf("redis delete: %w", err)
	}

	return nil
}

func (c *RedisCache) cacheTTL(url model.URL) time.Duration {
	if url.ExpiresAt == nil {
		return c.ttl
	}

	remaining := time.Until(*url.ExpiresAt)

	if remaining <= 0 {
		return time.Second
	}

	if remaining < c.ttl {
		return remaining
	}

	return c.ttl
}

func urlCacheKey(shortCode string) string {
	return "goshort:url:" + shortCode
}
