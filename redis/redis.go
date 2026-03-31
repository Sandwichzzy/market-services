package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Config 是 Redis 客户端的配置参数
type Config struct {
	Address  string // Redis 服务器地址，例如 "localhost:6379"
	Password string // Redis 认证密码，若无则留空
	DB       int    // 使用的数据库编号，默认 0
}

// Client 封装了 Redis 客户端，提供常用操作方法
type Client struct {
	rdb *redis.Client // 底层 go-redis 客户端
}

// New 创建并初始化 Redis 客户端，同时执行 PING 验证连接是否成功
// 如果连接失败则返回错误
func New(cfg Config) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.Address,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     50,              // 连接池大小
		MinIdleConns: 10,              // 最小空闲连接数
		DialTimeout:  5 * time.Second, // 连接超时
		ReadTimeout:  3 * time.Second, // 读取超时
		WriteTimeout: 3 * time.Second, // 写入超时
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return &Client{rdb: rdb}, nil
}

// Set 设置键值对，可指定过期时间（过期时间 <= 0 表示永不过期）
func (c *Client) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	return c.rdb.Set(ctx, key, value, expiration).Err()
}

// Get 获取指定键的值，若键不存在则返回 redis.Nil 错误
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	return c.rdb.Get(ctx, key).Result()
}

// Del 删除一个或多个键，返回删除的键数量（错误通过 error 返回）
func (c *Client) Del(ctx context.Context, keys ...string) error {
	return c.rdb.Del(ctx, keys...).Err()
}

// Exists 检查键是否存在，返回布尔值
func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	n, err := c.rdb.Exists(ctx, key).Result()
	return n > 0, err
}

// Incr 将指定键的值原子性自增 1，并返回自增后的值
func (c *Client) Incr(ctx context.Context, key string) (int64, error) {
	return c.rdb.Incr(ctx, key).Result()
}

// SMembers 返回集合（Set）中的所有成员
func (c *Client) SMembers(ctx context.Context, key string) ([]string, error) {
	return c.rdb.SMembers(ctx, key).Result()
}

// HSet 设置哈希表中的一个或多个字段值对
func (c *Client) HSet(ctx context.Context, key string, values ...interface{}) error {
	return c.rdb.HSet(ctx, key, values...).Err()
}

// HGetAll 获取哈希表中所有字段和值
func (c *Client) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return c.rdb.HGetAll(ctx, key).Result()
}

// ZAdd 向有序集合添加一个成员，并指定分数
func (c *Client) ZAdd(ctx context.Context, key string, score float64, member any) error {
	return c.rdb.ZAdd(ctx, key, redis.Z{
		Score:  score,
		Member: member,
	}).Err()
}

// ZRevRange 按分数从高到低返回有序集合中指定索引范围内的成员
func (c *Client) ZRevRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return c.rdb.ZRevRange(ctx, key, start, stop).Result()
}

// TryLock 尝试获取分布式锁（基于 SETNX），成功返回 true，失败返回 false
// key: 锁的键名；value: 锁的值（通常用于解锁时验证）；expiration: 锁的自动释放时间
func (c *Client) TryLock(ctx context.Context, key string, value string, expiration time.Duration) (bool, error) {
	return c.rdb.SetNX(ctx, key, value, expiration).Result()
}

// Unlock 释放分布式锁，即删除锁对应的键
func (c *Client) Unlock(ctx context.Context, key string) error {
	return c.rdb.Del(ctx, key).Err()
}

// Pipeline 返回一个 Redis 管道，用于批量执行命令
func (c *Client) Pipeline() redis.Pipeliner {
	return c.rdb.Pipeline()
}

// Publish 发布消息到指定频道（用于发布/订阅模式）
func (c *Client) Publish(ctx context.Context, channel string, message interface{}) error {
	return c.rdb.Publish(ctx, channel, message).Err()
}

// Subscribe 订阅一个或多个频道，返回 PubSub 对象用于接收消息
func (c *Client) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return c.rdb.Subscribe(ctx, channels...)
}
