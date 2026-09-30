package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// CleanupStaleRunningTasks Olası kalıntı running task keys'lerini temizle
// Bu, olağandışı durumlar nedeniyle kalan running keys'leri temizlemek için kullanılabilen bir hata ayıklama ve bakım aracıdır
func CleanupStaleRunningTasks(ctx context.Context, redisClient *redis.Client, keyPrefix string, maxAge time.Duration) (int, error) {
	// Eşleşen tüm keys'leri al
	keys, err := redisClient.Keys(ctx, keyPrefix+"*").Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get keys: %w", err)
	}

	if len(keys) == 0 {
		return 0, nil
	}

	// Her key'in TTL'sini kontrol et
	var staleTasks []string
	for _, key := range keys {
		ttl, err := redisClient.TTL(ctx, key).Result()
		if err != nil {
			continue // Hatalı key'leri atla
		}

		// TTL 0'dan küçükse (asla sona ermez) veya kalan süre çok uzunsa (muhtemelen kalıntıdır), stale olarak işaretle
		if ttl < 0 || ttl > maxAge {
			staleTasks = append(staleTasks, key)
		}
	}

	if len(staleTasks) == 0 {
		return 0, nil
	}

	// Stale keys'leri sil
	deleted, err := redisClient.Del(ctx, staleTasks...).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to delete stale keys: %w", err)
	}

	return int(deleted), nil
}

// CheckRunningTaskStatus belirtilen running task durumunu kontrol eder
func CheckRunningTaskStatus(ctx context.Context, redisClient *redis.Client, runningKey, progressKey string) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// running key kontrol edilir
	runningTaskID, err := redisClient.Get(ctx, runningKey).Result()
	if err != nil {
		if err == redis.Nil {
			result["running_task_exists"] = false
		} else {
			return nil, fmt.Errorf("failed to get running task: %w", err)
		}
	} else {
		result["running_task_exists"] = true
		result["running_task_id"] = runningTaskID

		// running key TTL'si alınır
		ttl, _ := redisClient.TTL(ctx, runningKey).Result()
		result["running_task_ttl"] = ttl.String()
	}

	// progress key kontrol edilir
	progressData, err := redisClient.Get(ctx, progressKey).Result()
	if err != nil {
		if err == redis.Nil {
			result["progress_exists"] = false
		} else {
			return nil, fmt.Errorf("failed to get progress: %w", err)
		}
	} else {
		result["progress_exists"] = true
		result["progress_data"] = progressData

		// progress key TTL'si alınır
		ttl, _ := redisClient.TTL(ctx, progressKey).Result()
		result["progress_ttl"] = ttl.String()
	}

	return result, nil
}
