package helpers

import (
	"sync"
	"time"
)

const MaxDownloads = 100        
const RateLimitWindow = 10 * time.Minute

var (
	mu           sync.Mutex
	downloadTimestamps []time.Time 
)

func IsRateLimitExceeded() bool {
	mu.Lock()
	defer mu.Unlock()

	now := time.Now()

	for len(downloadTimestamps) > 0 && now.Sub(downloadTimestamps[0]) > RateLimitWindow {
		downloadTimestamps = downloadTimestamps[1:]
	}

	if len(downloadTimestamps) >= MaxDownloads {
		return true
	}

	downloadTimestamps = append(downloadTimestamps, now)

	return false
}
