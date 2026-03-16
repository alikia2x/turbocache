package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Token          string
	CacheDirectory string
	Port           string

	// LRU eviction settings (0 = disabled)
	MaxCacheSize  int64 // max cache size in MB, 0 = disabled
	MaxCacheCount int   // max number of artifacts, 0 = disabled
	EvictBatch    int   // number of artifacts to evict per cleanup
}

func Load() *Config {
	_ = godotenv.Load()

	token := os.Getenv("TURBO_TOKEN")
	if token == "" {
		token = os.Getenv("TOKEN")
	}

	cacheDir := os.Getenv("CACHE_DIRECTORY")
	if cacheDir == "" {
		cacheDir = "./cache"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	maxCacheSize := parseMB(os.Getenv("MAX_CACHE_SIZE"))
	maxCacheCount := parseInt(os.Getenv("MAX_CACHE_COUNT"), 0)
	evictBatch := parseInt(os.Getenv("EVICT_BATCH"), 10)

	return &Config{
		Token:          token,
		CacheDirectory: cacheDir,
		Port:           port,
		MaxCacheSize:   maxCacheSize,
		MaxCacheCount:  maxCacheCount,
		EvictBatch:     evictBatch,
	}
}

func parseMB(s string) int64 {
	if s == "" {
		return 0
	}
	var n int64
	_, _ = fmt.Sscanf(s, "%d", &n)
	return n
}

func parseInt(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	var n int
	_, _ = fmt.Sscanf(s, "%d", &n)
	return n
}
