package application

import (
	"sync"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

type workLimiter struct {
	buckets map[string]map[int64]int
	mu      sync.Mutex
}

const workRateWindowSeconds = int64(time.Minute / time.Second)

func (l *workLimiter) allow(userID string, begin bool, now time.Time) error {
	if userID == "" {
		return domain.ErrBadRequest
	}
	window := now.Unix() / workRateWindowSeconds
	limit := domain.OtherWorkPerMinute
	class := "other:"
	if begin {
		limit = domain.BeginWorkPerMinute
		class = "begin:"
	}
	key := class + userID
	l.mu.Lock()
	defer l.mu.Unlock()
	for bucketKey, windows := range l.buckets {
		for bucket := range windows {
			if bucket != window {
				delete(windows, bucket)
			}
		}
		if len(windows) == 0 {
			delete(l.buckets, bucketKey)
		}
	}
	if l.buckets[key] == nil {
		l.buckets[key] = map[int64]int{}
	}
	if l.buckets[key][window] >= limit {
		return domain.ErrRateLimited
	}
	l.buckets[key][window]++
	return nil
}
