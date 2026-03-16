package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"

	"turbocache/models"
)

type Storage struct {
	cacheDir     string
	mu           sync.RWMutex
	evictMu      sync.Mutex
	evictRunning atomic.Bool
	maxSizeMB    int64
	maxCount     int
	evictBatch   int
	evictEnabled bool
}

func New(cacheDir string) *Storage {
	return &Storage{
		cacheDir:   cacheDir,
		evictBatch: 10,
	}
}

func (s *Storage) SetEvictionConfig(maxSizeMB int64, maxCount int, batch int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.maxSizeMB = maxSizeMB
	s.maxCount = maxCount
	s.evictBatch = batch
	s.evictEnabled = (maxSizeMB > 0 || maxCount > 0)
}

func (s *Storage) EvictionEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.evictEnabled
}

func (s *Storage) EnsureDir() error {
	return os.MkdirAll(s.cacheDir, 0755)
}

func (s *Storage) artifactPath(hash string) string {
	return filepath.Join(s.cacheDir, hash)
}

func (s *Storage) ArtifactPath(hash string) string {
	return s.artifactPath(hash)
}

func (s *Storage) metadataPath(hash string) string {
	return filepath.Join(s.cacheDir, hash+".meta")
}

func (s *Storage) Exists(hash string) (bool, error) {
	_, err := os.Stat(s.artifactPath(hash))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *Storage) Stat(hash string) (os.FileInfo, error) {
	return os.Stat(s.artifactPath(hash))
}

func (s *Storage) GetMetadata(hash string) (*models.ArtifactMetadata, error) {
	data, err := os.ReadFile(s.metadataPath(hash))
	if err != nil {
		return nil, err
	}
	var meta models.ArtifactMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func (s *Storage) SaveMetadata(hash string, meta *models.ArtifactMetadata) error {
	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return os.WriteFile(s.metadataPath(hash), data, 0600)
}

func (s *Storage) Save(hash string, data []byte, meta *models.ArtifactMetadata) error {
	if err := os.WriteFile(s.artifactPath(hash), data, 0600); err != nil {
		return err
	}
	if meta != nil {
		if err := s.SaveMetadata(hash, meta); err != nil {
			return err
		}
	}
	go s.TryEvict()
	return nil
}

func (s *Storage) TryEvict() {
	if !s.EvictionEnabled() {
		return
	}

	// Prevent concurrent eviction runs
	if !s.evictRunning.CompareAndSwap(false, true) {
		return
	}
	defer s.evictRunning.Store(false)

	s.evictMu.Lock()
	defer s.evictMu.Unlock()

	s.mu.RLock()
	maxSize := s.maxSizeMB
	maxCount := s.maxCount
	batch := s.evictBatch
	s.mu.RUnlock()

	_, err := s.EvictLRU(maxSize, maxCount, batch)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARNING: LRU eviction error: %v\n", err)
	}
}

func (s *Storage) Get(hash string) ([]byte, error) {
	return os.ReadFile(s.artifactPath(hash))
}

func (s *Storage) Query(hashes []string) map[string]interface{} {
	result := make(map[string]interface{})
	for _, hash := range hashes {
		path := s.artifactPath(hash)
		info, err := os.Stat(path)
		if err != nil {
			result[hash] = nil
			continue
		}

		meta, err := s.GetMetadata(hash)
		if err != nil {
			result[hash] = models.ArtifactInfo{
				Size:           info.Size(),
				TaskDurationMs: 0,
			}
			continue
		}

		result[hash] = models.ArtifactInfo{
			Size:           meta.Size,
			TaskDurationMs: meta.TaskDurationMs,
			Tag:            meta.Tag,
		}
	}
	return result
}

func (s *Storage) ParseDurationHeader(header string) int64 {
	var duration int64
	if header != "" {
		_, _ = fmt.Sscanf(header, "%d", &duration)
	}
	return duration
}

type artifactEntry struct {
	hash    string
	size    int64
	modTime int64
}

func (s *Storage) GetAllArtifacts() ([]artifactEntry, error) {
	// Release lock before I/O to avoid blocking other operations
	return s.scanArtifactsLocked()
}

func (s *Storage) GetCacheStats() (totalSize int64, count int, err error) {
	entries, err := s.GetAllArtifacts()
	if err != nil {
		return 0, 0, err
	}

	totalSize = 0
	for _, e := range entries {
		totalSize += e.size
	}
	return totalSize, len(entries), nil
}

func (s *Storage) EvictLRU(maxSizeMB int64, maxCount int, batch int) (evicted int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := s.scanArtifactsLocked()
	if err != nil {
		return 0, err
	}

	if len(entries) == 0 {
		return 0, nil
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].modTime < entries[j].modTime
	})

	totalSize := int64(0)
	for _, e := range entries {
		totalSize += e.size
	}

	targetCount := maxCount
	targetSize := maxSizeMB * 1024 * 1024

	for _, e := range entries {
		if targetCount > 0 && len(entries) <= targetCount {
			break
		}
		if targetSize > 0 && totalSize <= targetSize {
			break
		}

		if err := s.deleteArtifactLocked(e.hash); err != nil {
			continue
		}

		totalSize -= e.size
		evicted++
		entries = entries[1:]

		if evicted >= batch {
			break
		}
	}

	return evicted, nil
}

func (s *Storage) scanArtifactsLocked() ([]artifactEntry, error) {
	entries := []artifactEntry{}

	dir, err := os.Open(s.cacheDir)
	if err != nil {
		return nil, err
	}
	defer dir.Close()

	files, err := dir.Readdir(-1)
	if err != nil {
		return nil, err
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}
		name := f.Name()
		if len(name) > 4 && name[len(name)-4:] == ".meta" {
			continue
		}
		entries = append(entries, artifactEntry{
			hash:    name,
			size:    f.Size(),
			modTime: f.ModTime().UnixNano(),
		})
	}

	return entries, nil
}

func (s *Storage) deleteArtifactLocked(hash string) error {
	artifactPath := s.artifactPath(hash)
	metaPath := s.metadataPath(hash)

	if err := os.Remove(artifactPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Remove(metaPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (s *Storage) Delete(hash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.deleteArtifactLocked(hash)
}
