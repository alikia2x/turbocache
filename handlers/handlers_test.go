package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"turbocache/models"
	"turbocache/storage"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	return r
}

func TestGetArtifactStatus(t *testing.T) {
	r := setupTestRouter()
	tmpDir := t.TempDir()
	s := storage.New(tmpDir)
	h := New(s)
	r.GET("/artifacts/status", h.GetArtifactStatus)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/artifacts/status", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp models.CachingStatusResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "enabled", resp.Status)
}

func TestArtifactExists_NotFound(t *testing.T) {
	r := setupTestRouter()
	tmpDir := t.TempDir()
	s := storage.New(tmpDir)
	h := New(s)
	r.HEAD("/artifacts/:hash", h.ArtifactExists)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("HEAD", "/artifacts/nonexistent", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestArtifactExists_Success(t *testing.T) {
	r := setupTestRouter()
	tmpDir := t.TempDir()
	s := storage.New(tmpDir)
	_ = s.Save("testhash", []byte("data"), &models.ArtifactMetadata{Size: 4, TaskDurationMs: 100})
	h := New(s)
	r.HEAD("/artifacts/:hash", h.ArtifactExists)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("HEAD", "/artifacts/testhash", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "4", w.Header().Get("Content-Length"))
	assert.Equal(t, "100", w.Header().Get("X-Artifact-Duration"))
}

func TestDownloadArtifact_NotFound(t *testing.T) {
	r := setupTestRouter()
	tmpDir := t.TempDir()
	s := storage.New(tmpDir)
	h := New(s)
	r.GET("/artifacts/:hash", h.DownloadArtifact)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/artifacts/nonexistent", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "ARTIFACT_NOT_FOUND", resp.Code)
}

func TestDownloadArtifact_Success(t *testing.T) {
	r := setupTestRouter()
	tmpDir := t.TempDir()
	s := storage.New(tmpDir)
	_ = s.Save("testhash", []byte("test data"), &models.ArtifactMetadata{Size: 9, TaskDurationMs: 50, Tag: "v1.0"})
	h := New(s)
	r.GET("/artifacts/:hash", h.DownloadArtifact)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/artifacts/testhash", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "9", w.Header().Get("Content-Length"))
	assert.Equal(t, "50", w.Header().Get("X-Artifact-Duration"))
	assert.Equal(t, "v1.0", w.Header().Get("X-Artifact-Tag"))
	assert.Equal(t, "application/octet-stream", w.Header().Get("Content-Type"))
	assert.Equal(t, "test data", w.Body.String())
}

func TestUploadArtifact_Success(t *testing.T) {
	r := setupTestRouter()
	tmpDir := t.TempDir()
	s := storage.New(tmpDir)
	h := New(s)
	r.PUT("/artifacts/:hash", h.UploadArtifact)

	body := bytes.NewBufferString("artifact content")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/artifacts/newhash", body)
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("X-Artifact-Duration", "200")
	req.Header.Set("X-Artifact-Tag", "v2.0")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp models.ArtifactUploadResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, []string{"/v8/artifacts/newhash"}, resp.Urls)

	exists, err := s.Exists("newhash")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestUploadArtifact_ReadError(t *testing.T) {
	r := setupTestRouter()
	tmpDir := t.TempDir()
	s := storage.New(tmpDir)
	h := New(s)
	r.PUT("/artifacts/:hash", h.UploadArtifact)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/artifacts/test", &errorReader{})
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestQueryArtifacts_InvalidBody(t *testing.T) {
	r := setupTestRouter()
	tmpDir := t.TempDir()
	s := storage.New(tmpDir)
	h := New(s)
	r.POST("/artifacts", h.QueryArtifacts)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/artifacts", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestQueryArtifacts_Success(t *testing.T) {
	r := setupTestRouter()
	tmpDir := t.TempDir()
	s := storage.New(tmpDir)
	_ = s.Save("hash1", []byte("data1"), &models.ArtifactMetadata{Size: 5, TaskDurationMs: 100})
	h := New(s)
	r.POST("/artifacts", h.QueryArtifacts)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/artifacts", bytes.NewBufferString(`{"hashes":["hash1","hash2"]}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.NotNil(t, resp["hash1"])
	assert.Nil(t, resp["hash2"])
}

func TestRecordCacheEvents_InvalidBody(t *testing.T) {
	r := setupTestRouter()
	tmpDir := t.TempDir()
	s := storage.New(tmpDir)
	h := New(s)
	r.POST("/artifacts/events", h.RecordCacheEvents)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/artifacts/events", nil)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "BAD_REQUEST", resp.Code)
}

func TestRecordCacheEvents_ValidBody(t *testing.T) {
	r := setupTestRouter()
	tmpDir := t.TempDir()
	s := storage.New(tmpDir)
	h := New(s)
	r.POST("/artifacts/events", h.RecordCacheEvents)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/artifacts/events", bytes.NewBufferString(`[{"sessionId":"test","source":"LOCAL","event":"HIT","hash":"abc123"}]`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, assert.AnError
}

func (e *errorReader) Close() error {
	return nil
}

var _ io.ReadCloser = (*errorReader)(nil)
