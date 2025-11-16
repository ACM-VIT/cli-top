package helpers

import (
	"sync"
	"time"

	types "cli-top/types"
)

const semesterCacheTTL = 15 * time.Minute

var (
	semCacheMu sync.RWMutex
	semCache   = make(map[string]semesterCacheEntry)
)

type semesterCacheEntry struct {
	semesters []types.Semester
	cachedAt  time.Time
}

func cloneSemesters(src []types.Semester) []types.Semester {
	cloned := make([]types.Semester, len(src))
	copy(cloned, src)
	return cloned
}

func getCachedSemesters(regNo string) ([]types.Semester, bool) {
	semCacheMu.RLock()
	entry, ok := semCache[regNo]
	semCacheMu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Since(entry.cachedAt) > semesterCacheTTL {
		semCacheMu.Lock()
		delete(semCache, regNo)
		semCacheMu.Unlock()
		return nil, false
	}
	return cloneSemesters(entry.semesters), true
}

func storeSemesters(regNo string, semesters []types.Semester) {
	semCacheMu.Lock()
	semCache[regNo] = semesterCacheEntry{semesters: cloneSemesters(semesters), cachedAt: time.Now()}
	semCacheMu.Unlock()
}

// InvalidateSemesterCache removes cached semester data for the provided reg no.
func InvalidateSemesterCache(regNo string) {
	semCacheMu.Lock()
	delete(semCache, regNo)
	semCacheMu.Unlock()
}
