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

	// 期限切れのログ削除
	if err := s.removeOutdatedLogs(ctx, identifier, now); err != nil {
		return false, fmt.Errorf("failed to remove outdated logs, detail = %w", err)
	}

	// 新しいリクエストのログ追加
	member := redis.Z{
		Score:  float64(now.UnixNano()),
		Member: float64(now.UnixNano()),
	}
	s.Client.ZAdd(ctx, identifier, member)

	// リクエスト制限数以内か判定
	// 期限切れのログは全てｚ削除ｚなので全件取得すればOK
	if logs, err := s.getAllLogs(ctx, identifier); err != nil {
		return false, fmt.Errorf("failed to get logs. detail = %w", err)
	} else if len(logs) > s.Limit {
		fmt.Printf("key = %s, count = %d\n", identifier, len(logs))
		return false, nil
	}

	return true, nil
}

func (s *slidingWindowLogRedisStore) getAllLogs(ctx context.Context, identifier string) ([]string, error) {
	return s.Client.ZRange(ctx, identifier, 0, -1).Result()
}

func (s *slidingWindowLogRedisStore) removeOutdatedLogs(ctx context.Context, identifier string, now time.Time) error {
	outdatedMax := now.Add(-time.Duration(s.WindowSize) * time.Second).UnixNano()
	_, err := s.Client.ZRemRangeByScore(ctx, identifier, "0", strconv.Itoa(int(outdatedMax))).Result()
	return err
}
