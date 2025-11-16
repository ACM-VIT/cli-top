package helpers

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/lpernett/godotenv"
	"github.com/spf13/viper"

	types "cli-top/types"
)

const (
	semesterCacheTTL    = 15 * time.Minute
	semesterCacheEnvKey = "SEMESTER_CACHE"
)

var (
	semCacheMu sync.RWMutex
	semCache   = make(map[string]semesterCacheEntry)
)

type semesterCacheEntry struct {
	semesters []types.Semester
	cachedAt  time.Time
}

type persistedSemesterCacheEntry struct {
	CachedAt  string           `json:"cachedAt"`
	Semesters []types.Semester `json:"semesters"`
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
		persistSemesterCache()
		return nil, false
	}
	return cloneSemesters(entry.semesters), true
}

func storeSemesters(regNo string, semesters []types.Semester) {
	semCacheMu.Lock()
	semCache[regNo] = semesterCacheEntry{semesters: cloneSemesters(semesters), cachedAt: time.Now()}
	semCacheMu.Unlock()
	persistSemesterCache()
}

// InvalidateSemesterCache removes cached semester data for the provided reg no.
func InvalidateSemesterCache(regNo string) {
	semCacheMu.Lock()
	delete(semCache, regNo)
	semCacheMu.Unlock()
	persistSemesterCache()
}

// LoadSemesterCacheFromEnv hydrates the in-memory cache from the persisted value
// stored in the config file / environment variable.
func LoadSemesterCacheFromEnv() {
	encoded := os.Getenv(semesterCacheEnvKey)
	if encoded == "" {
		return
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return
	}
	var payload map[string]persistedSemesterCacheEntry
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return
	}
	semCacheMu.Lock()
	defer semCacheMu.Unlock()
	semCache = make(map[string]semesterCacheEntry)
	for regNo, entry := range payload {
		cachedAt, err := time.Parse(time.RFC3339Nano, entry.CachedAt)
		if err != nil {
			continue
		}
		semCache[regNo] = semesterCacheEntry{
			semesters: cloneSemesters(entry.Semesters),
			cachedAt:  cachedAt,
		}
	}
}

func persistSemesterCache() {
	semCacheMu.RLock()
	snapshot := make(map[string]persistedSemesterCacheEntry, len(semCache))
	for regNo, entry := range semCache {
		if time.Since(entry.cachedAt) > semesterCacheTTL {
			continue
		}
		snapshot[regNo] = persistedSemesterCacheEntry{
			CachedAt:  entry.cachedAt.Format(time.RFC3339Nano),
			Semesters: cloneSemesters(entry.semesters),
		}
	}
	semCacheMu.RUnlock()

	data, err := json.Marshal(snapshot)
	if err != nil {
		return
	}
	encoded := base64.StdEncoding.EncodeToString(data)
	os.Setenv(semesterCacheEnvKey, encoded)
	viper.Set(semesterCacheEnvKey, "\""+encoded+"\"")
	writeEnvValue(semesterCacheEnvKey, encoded)
}

func writeEnvValue(key, value string) {
	configPath := ConfigFilePath()
	values, err := godotenv.Read(configPath)
	if err != nil {
		values = make(map[string]string)
	}
	values[key] = value
	_ = godotenv.Write(values, configPath)
}
