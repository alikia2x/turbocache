package storage

import (
	"os"
	"path/filepath"
	"testing"

	"turbocache/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorage_New(t *testing.T) {
	s := New("/tmp/test-cache")
	assert.Equal(t, "/tmp/test-cache", s.cacheDir)
}

func TestStorage_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	s := New(tmpDir)

	hash := "test-hash-123"
	data := []byte("test artifact data")
	meta := &models.ArtifactMetadata{
		Size:           20,
		TaskDurationMs: 1000,
		Tag:            "v1.0.0",
	}

	err := s.Save(hash, data, meta)
	require.NoError(t, err)

	loadedData, err := s.Get(hash)
	require.NoError(t, err)
	assert.Equal(t, data, loadedData)

	loadedMeta, err := s.GetMetadata(hash)
	require.NoError(t, err)
	assert.Equal(t, meta.Size, loadedMeta.Size)
	assert.Equal(t, meta.TaskDurationMs, loadedMeta.TaskDurationMs)
	assert.Equal(t, meta.Tag, loadedMeta.Tag)
}

func TestStorage_Exists(t *testing.T) {
	tmpDir := t.TempDir()
	s := New(tmpDir)

	hash := "test-hash"
	exists, err := s.Exists(hash)
	assert.NoError(t, err)
	assert.False(t, exists)

	err = s.Save(hash, []byte("data"), nil)
	require.NoError(t, err)

	exists, err = s.Exists(hash)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestStorage_Query(t *testing.T) {
	tmpDir := t.TempDir()
	s := New(tmpDir)

	_ = s.Save("hash1", []byte("data1"), &models.ArtifactMetadata{Size: 6, TaskDurationMs: 100})
	_ = s.Save("hash2", []byte("data2"), &models.ArtifactMetadata{Size: 6, TaskDurationMs: 200})
	_ = s.Save("hash3", []byte("data3"), nil)

	result := s.Query([]string{"hash1", "hash2", "hash3", "hash4"})

	info1 := result["hash1"].(models.ArtifactInfo)
	assert.Equal(t, int64(6), info1.Size)
	assert.Equal(t, int64(100), info1.TaskDurationMs)

	info2 := result["hash2"].(models.ArtifactInfo)
	assert.Equal(t, int64(6), info2.Size)
	assert.Equal(t, int64(200), info2.TaskDurationMs)

	info3 := result["hash3"].(models.ArtifactInfo)
	assert.Equal(t, int64(5), info3.Size)
	assert.Equal(t, int64(0), info3.TaskDurationMs)

	assert.Nil(t, result["hash4"])
}

func TestStorage_EnsureDir(t *testing.T) {
	tmpDir := t.TempDir()
	s := New(filepath.Join(tmpDir, "subdir", "nested"))

	err := s.EnsureDir()
	require.NoError(t, err)

	_, err = os.Stat(s.ArtifactPath("."))
	assert.NoError(t, err)
}
