package handlers

import (
	"fmt"
	"io"
	"net/http"

	"turbocache/models"
	"turbocache/storage"

	"github.com/gin-gonic/gin"
)

type Handlers struct {
	storage *storage.Storage
}

func New(s *storage.Storage) *Handlers {
	return &Handlers{storage: s}
}

func (h *Handlers) GetArtifactStatus(c *gin.Context) {
	c.JSON(http.StatusOK, models.CachingStatusResponse{
		Status: "enabled",
	})
}

func (h *Handlers) ArtifactExists(c *gin.Context) {
	hash := c.Param("hash")

	exists, err := h.storage.Exists(hash)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to check artifact",
		})
		return
	}

	if !exists {
		c.Status(http.StatusNotFound)
		return
	}

	info, err := h.storage.Stat(hash)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to stat artifact",
		})
		return
	}

	meta, err := h.storage.GetMetadata(hash)
	if err != nil {
		c.Header("Content-Length", fmt.Sprintf("%d", info.Size()))
		c.Status(http.StatusOK)
		return
	}

	c.Header("Content-Length", fmt.Sprintf("%d", meta.Size))
	if meta.TaskDurationMs > 0 {
		c.Header("X-Artifact-Duration", fmt.Sprintf("%d", meta.TaskDurationMs))
	}
	c.Status(http.StatusOK)
}

func (h *Handlers) DownloadArtifact(c *gin.Context) {
	hash := c.Param("hash")

	exists, err := h.storage.Exists(hash)
	if err != nil || !exists {
		c.AbortWithStatusJSON(http.StatusNotFound, models.ErrorResponse{
			Code:    "ARTIFACT_NOT_FOUND",
			Message: "Artifact not found",
		})
		return
	}

	info, err := h.storage.Stat(hash)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to read artifact",
		})
		return
	}

	meta, _ := h.storage.GetMetadata(hash)

	c.Header("Content-Length", fmt.Sprintf("%d", info.Size()))
	if meta != nil {
		if meta.TaskDurationMs > 0 {
			c.Header("X-Artifact-Duration", fmt.Sprintf("%d", meta.TaskDurationMs))
		}
		if meta.Tag != "" {
			c.Header("X-Artifact-Tag", meta.Tag)
		}
	}
	c.Header("Content-Type", "application/octet-stream")
	c.File(h.storage.ArtifactPath(hash))
}

func (h *Handlers) UploadArtifact(c *gin.Context) {
	hash := c.Param("hash")

	if err := h.storage.EnsureDir(); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to create cache directory",
		})
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    "BAD_REQUEST",
			Message: "Failed to read request body",
		})
		return
	}

	durationMs := h.storage.ParseDurationHeader(c.GetHeader("X-Artifact-Duration"))
	tag := c.GetHeader("X-Artifact-Tag")

	meta := &models.ArtifactMetadata{
		Size:           int64(len(body)),
		TaskDurationMs: durationMs,
		Tag:            tag,
	}

	if err := h.storage.Save(hash, body, meta); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to save artifact",
		})
		return
	}

	c.JSON(http.StatusOK, models.ArtifactUploadResponse{
		Urls: []string{"/v8/artifacts/" + hash},
	})
}

func (h *Handlers) QueryArtifacts(c *gin.Context) {
	var req models.ArtifactQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    "BAD_REQUEST",
			Message: "Invalid request body",
		})
		return
	}

	response := h.storage.Query(req.Hashes)
	c.JSON(http.StatusOK, response)
}

func (h *Handlers) RecordCacheEvents(c *gin.Context) {
	var events []models.CacheEvent
	if err := c.ShouldBindJSON(&events); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    "BAD_REQUEST",
			Message: "Invalid request body",
		})
		return
	}

	c.Status(http.StatusOK)
}
