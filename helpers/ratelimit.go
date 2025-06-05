package helpers

import (
	"sync"
	"time"
)

const MaxDownloads = 100
const RateLimitWindow = 10 * time.Minute

var (
	mu       sync.Mutex
	tsBuffer [MaxDownloads]time.Time
	start    int
	count    int
)

func IsRateLimitExceeded() bool {
	mu.Lock()
	defer mu.Unlock()

	now := time.Now()

	// drop all timestamps outside the rate limit window
	for count > 0 {
		oldest := tsBuffer[start]
		if now.Sub(oldest) <= RateLimitWindow {
			break
		}
		start = (start + 1) % MaxDownloads
		count--
	}

	if count >= MaxDownloads {
		return true
	}

	insertIdx := (start + count) % MaxDownloads
	tsBuffer[insertIdx] = now
	if count < MaxDownloads {
		count++
	}

	return false
}
