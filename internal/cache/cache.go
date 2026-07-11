package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	client *redis.Client
}

func NewCache(addr string) *Cache {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	return &Cache{client: client}
}

func (c *Cache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

func (c *Cache) GetExchangeRates(ctx context.Context, baseCurrency string) (map[string]float64, error) {
	key := fmt.Sprintf("rates:%s", baseCurrency)

	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}

	var rates map[string]float64
	if err := json.Unmarshal(data, &rates); err != nil {
		return nil, err
	}

	return rates, nil
}

func (c *Cache) SetExchangeRates(ctx context.Context, baseCurrency string, rates map[string]float64) error {
	key := fmt.Sprintf("rates:%s", baseCurrency)

	data, err := json.Marshal(rates)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, data, 1*time.Hour).Err()
}

func (c *Cache) InvalidateRates(ctx context.Context, baseCurrency string) error {
	key := fmt.Sprintf("rates:%s", baseCurrency)
	return c.client.Del(ctx, key).Err()
}

func (c *Cache) IncrFailedAttempts(ctx context.Context, email string) (int64, error) {
	key := fmt.Sprintf("failed_attempts:%s", email)
	count, err := c.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	
	c.client.Expire(ctx, key, 30*time.Minute)
	return count, nil
}

func (c *Cache) LockAccount(ctx context.Context, email string) error {
	key := fmt.Sprintf("locked:%s", email)
	return c.client.Set(ctx, key, true, 30*time.Minute).Err()
}

func (c *Cache) IsAccountLocked(ctx context.Context, email string) (bool, error) {
	key := fmt.Sprintf("locked:%s", email)
	exists, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

func (c *Cache) ResetFailedAttempts(ctx context.Context, email string) error {
	key := fmt.Sprintf("failed_attempts:%s", email)
	return c.client.Del(ctx, key).Err()
}