package main

import (
	"fmt"
	"os"

	"turbocache/config"
	"turbocache/handlers"
	"turbocache/middleware"
	"turbocache/storage"

	"github.com/gin-gonic/gin"
)

const (
	colorYellow = "\033[33m"
	colorGreen  = "\033[32m"
	colorRed    = "\033[31m"
	colorReset  = "\033[0m"
)

func printWarning(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, colorYellow+"WARNING: "+colorReset+format+"\n", args...)
}

func printInfo(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

func main() {
	cfg := config.Load()

	s := storage.New(cfg.CacheDirectory)
	s.SetEvictionConfig(cfg.MaxCacheSize, cfg.MaxCacheCount, cfg.EvictBatch)
	if err := s.EnsureDir(); err != nil {
		printWarning("failed to create cache directory: %v", err)
	}

	h := handlers.New(s)

	r := gin.Default()

	v8 := r.Group("/v8")
	v8.Use(middleware.Auth(cfg.Token))
	{
		v8.GET("/artifacts/status", h.GetArtifactStatus)
		v8.GET("/artifacts/:hash", h.DownloadArtifact)
		v8.HEAD("/artifacts/:hash", h.ArtifactExists)
		v8.PUT("/artifacts/:hash", h.UploadArtifact)
		v8.POST("/artifacts", h.QueryArtifacts)
		v8.POST("/artifacts/events", h.RecordCacheEvents)
	}

	fmt.Printf("Starting server on port %s\n", cfg.Port)
	fmt.Printf("Cache directory: %s\n", cfg.CacheDirectory)
	if cfg.Token != "" {
		printInfo("Token authentication %senabled%s", colorGreen, colorReset)
	} else {
		printWarning("No token set - authentication is disabled!")
		printInfo("To enable authentication, set TURBO_TOKEN or TOKEN environment variable:")
		printInfo("  export TURBO_TOKEN=your-secret-token")
		printInfo("  # Or create a .env file with: TURBO_TOKEN=your-secret-token")
	}

	if cfg.MaxCacheSize > 0 || cfg.MaxCacheCount > 0 {
		printInfo("LRU eviction %senabled%s", colorGreen, colorReset)
		if cfg.MaxCacheSize > 0 {
			printInfo("  Max cache size: %d MB", cfg.MaxCacheSize)
		}
		if cfg.MaxCacheCount > 0 {
			printInfo("  Max cache count: %d artifacts", cfg.MaxCacheCount)
		}
		printInfo("  Evict batch: %d", cfg.EvictBatch)
	}

	if err := r.Run(":" + cfg.Port); err != nil {
		printWarning("Server failed to start: %v", err)
		os.Exit(1)
	}
}
