package middleware

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// slidingWindowLogRedisStore Sliding Window Log リクエスト制限
type slidingWindowLogRedisStore struct {
	Client     *redis.Client
	WindowSize int // second
	Limit      int
}

func NewSlidingWindowLogRedisStore(client *redis.Client, window, limit int) *slidingWindowLogRedisStore {
	return &slidingWindowLogRedisStore{
		Client:     client,
		WindowSize: window,
		Limit:      limit,
	}
}

func (s *slidingWindowLogRedisStore) Allow(identifier string) (bool, error) {
	ctx := context.Background()
	now := time.Now()
	unixNow := now.UnixNano()
	_, txErr := s.Client.TxPipelined(ctx, func(p redis.Pipeliner) error {
		// 期限切れのログ削除
		outdatedMax := s.prevWindowTime(now).UnixNano()
		if _, err := p.ZRemRangeByScore(ctx, identifier, "0", strconv.Itoa(int(outdatedMax))).Result(); err != nil {
			return fmt.Errorf("failed to remove outdated logs, detail = %w", err)
		}

		// 新しいリクエストのログ追加
		member := redis.Z{
			Score:  float64(unixNow),
			Member: float64(unixNow),
		}
		if _, err := p.ZAdd(ctx, identifier, member).Result(); err != nil {
			return fmt.Errorf("failed to add log, detail = %w", err)
		}
		// expireを指定しておかないのアクセスしなくなったユーザーのログが残り続けてしまう
		if _, err := p.Expire(ctx, identifier, s.expiration()).Result(); err != nil {
			return fmt.Errorf("failed to expire, detail = %w", err)
		}

		return nil
	})

	if txErr != nil {
		return false, txErr
	}

	// リクエスト制限数以内か判定
	// 期限切れのログは削除しているので全件取得すればOK
	if logs, err := s.Client.ZRange(ctx, identifier, 0, unixNow).Result(); err != nil {
		return false, fmt.Errorf("failed to get logs. detail = %w", err)
	} else if len(logs) > s.Limit {
		fmt.Printf("[limited] key = %s, count = %d, unix = %d\n", identifier, len(logs), unixNow)
		return false, nil
	} else {
		fmt.Printf("[allow] key = %s, count = %d, unix = %d\n", identifier, len(logs), unixNow)
	}

	return true, nil
}

func (s *slidingWindowLogRedisStore) expiration() time.Duration {
	return time.Duration(s.WindowSize) * time.Second
}

func (s *slidingWindowLogRedisStore) prevWindowTime(now time.Time) time.Time {
	return now.Add(-time.Duration(s.WindowSize) * time.Second)
}
