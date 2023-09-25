package middleware

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// slidingWindowLogRedisStore Sliding Window Log リクエスト制限
// nano秒レベルで同一のリクエストが来なければ正確に制御できるはず
type slidingWindowLogRedisStore struct {
	Client    *redis.Client
	WindowSec time.Duration // second
	Limit     int
}

func NewSlidingWindowLogRedisStore(client *redis.Client, window time.Duration, limit int) *slidingWindowLogRedisStore {
	return &slidingWindowLogRedisStore{
		Client:    client,
		WindowSec: window,
		Limit:     limit,
	}
}

// Allow
// リクエスト日時(UnixNano)をredisの sorted set に保存
// 1. 前window期間のリクエスト日時は不要なので、ZREMRANGEBYSCOREで削除
// 2. ZADDでmember追加
// 3. expiration指定（window期間）指定しないとアクセスしなくなったユーザーのデータが消えない
// 4. ZRANGE でカレントリクエストの日時までのリクエスト数を取得
// 5. Limit以内か判定
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
		// expireを指定しておかないとアクセスしなくなったユーザーのログが残り続けてしまう
		if _, err := p.Expire(ctx, identifier, s.WindowSec).Result(); err != nil {
			return fmt.Errorf("failed to expire, detail = %w", err)
		}

		return nil
	})

	if txErr != nil {
		return false, txErr
	}

	// リクエスト制限数以内か判定
	// カレントリクエストの日時までのリクエスト数をカウント
	// 全件取得すると同時に複数のリクエストが来たときに正しく判定できないため
	// これでもnano秒レベルで同じだとダメなんだけど。
	if logs, err := s.Client.ZRange(ctx, identifier, 0, unixNow).Result(); err != nil {
		return false, fmt.Errorf("failed to get logs. detail = %w", err)
	} else if len(logs) > s.Limit {
		fmt.Printf("[limited] key = %s, count = %d, unix = %d\n", identifier, len(logs), unixNow)
		return false, nil
	} else {
		fmt.Printf("[allowed] key = %s, count = %d, unix = %d\n", identifier, len(logs), unixNow)
	}

	return true, nil
}

func (s *slidingWindowLogRedisStore) prevWindowTime(now time.Time) time.Time {
	return now.Add(-s.WindowSec)
}
