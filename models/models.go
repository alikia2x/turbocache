package models

// Response schemas from the OpenAPI spec

type CachingStatusResponse struct {
	Status string `json:"status"`
}

type ArtifactUploadResponse struct {
	Urls []string `json:"urls"`
}

type ArtifactQueryRequest struct {
	Hashes []string `json:"hashes"`
}

type ArtifactInfo struct {
	Size           int64  `json:"size"`
	TaskDurationMs int64  `json:"taskDurationMs"`
	Tag            string `json:"tag,omitempty"`
}

type ArtifactError struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

type CacheEvent struct {
	SessionId string `json:"sessionId"`
	Source    string `json:"source"`
	Event     string `json:"event"`
	Hash      string `json:"hash"`
	Duration  int64  `json:"duration,omitempty"`
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ArtifactMetadata - Metadata stored alongside artifacts
type ArtifactMetadata struct {
	Size           int64  `json:"size"`
	TaskDurationMs int64  `json:"taskDurationMs"`
	Tag            string `json:"tag"`
}
