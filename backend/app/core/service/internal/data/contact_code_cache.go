package data

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
)

const (
	// contactCodeKeyTTL 验证码会话存活时长。
	contactCodeKeyTTL = 10 * time.Minute

	// contactCodeKeyFormat 验证码会话键。对目标（手机号/邮箱）做 sha256，
	// 避免明文联系方式落入 redis 键。
	contactCodeKeyFormat = ProjectPrefix + "cc:%s"
)

// ContactCodeCache 存储联系方式验证码会话。
type ContactCodeCache struct {
	log *log.Helper
	rdb *redis.Client
}

func NewContactCodeCache(ctx *bootstrap.Context, rdb *redis.Client) *ContactCodeCache {
	return &ContactCodeCache{
		rdb: rdb,
		log: ctx.NewLoggerHelper("contact-code/cache/core-service"),
	}
}

func (c *ContactCodeCache) makeKey(dest string) string {
	sum := sha256.Sum256([]byte(dest))
	return fmt.Sprintf(contactCodeKeyFormat, hex.EncodeToString(sum[:]))
}

// Store 写入验证码会话，TTL 固定 contactCodeKeyTTL。
func (c *ContactCodeCache) Store(ctx context.Context, dest, code string) error {
	return c.rdb.Set(ctx, c.makeKey(dest), code, contactCodeKeyTTL).Err()
}

// Verify 校验验证码。命中即删除（单次使用）；未命中或不存在返回 false。
func (c *ContactCodeCache) Verify(ctx context.Context, dest, code string) bool {
	key := c.makeKey(dest)
	stored, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false
		}
		c.log.Error(err)
		return false
	}
	// 无论是否匹配，会话一次性消费。
	_ = c.rdb.Del(ctx, key).Err()
	return stored == code
}
