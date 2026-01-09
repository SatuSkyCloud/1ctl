package api

import (
	"1ctl/internal/context"
	"1ctl/internal/utils"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// StorageConfig represents a storage configuration
type StorageConfig struct {
	StorageID    uuid.UUID `json:"storage_id"`
	DeploymentID uuid.UUID `json:"deployment_id"`
	Namespace    string    `json:"namespace"`
	Name         string    `json:"name"`
	BucketName   string    `json:"bucket_name"`
	Type         string    `json:"type"`
	SizeBytes    int64     `json:"size_bytes"`
	UsedBytes    int64     `json:"used_bytes"`
	Endpoint     string    `json:"endpoint"`
	AccessKey    string    `json:"access_key,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// S3Bucket represents an S3 bucket
type S3Bucket struct {
	BucketName     string    `json:"bucket_name"`
	OrganizationID uuid.UUID `json:"organization_id"`
	ObjectCount    int       `json:"object_count"`
	SizeBytes      int64     `json:"size_bytes"`
	CreatedAt      time.Time `json:"created_at"`
}

// S3Object represents an S3 object/file
type S3Object struct {
	ObjectID     uuid.UUID `json:"object_id"`
	StorageID    uuid.UUID `json:"storage_id"`
	Key          string    `json:"key"`
	Size         int64     `json:"size"`
	ContentType  string    `json:"content_type"`
	LastModified time.Time `json:"last_modified"`
}

// StorageUsage represents storage usage statistics
type StorageUsage struct {
	StorageID   uuid.UUID `json:"storage_id"`
	TotalBytes  int64     `json:"total_bytes"`
	UsedBytes   int64     `json:"used_bytes"`
	ObjectCount int       `json:"object_count"`
}

// CreateStorageRequest represents a request to create storage
type CreateStorageRequest struct {
	DeploymentID uuid.UUID `json:"deployment_id"`
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	SizeBytes    int64     `json:"size_bytes"`
}

// PresignedURLResponse represents a presigned URL response
type PresignedURLResponse struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}

// GetStorageConfigs gets storage configurations for a namespace
func GetStorageConfigs(namespace string) ([]StorageConfig, error) {
	if namespace == "" {
		namespace = context.GetCurrentNamespace()
	}

	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/storage/namespace/%s", namespace), nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var configs []StorageConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal storage configs: %s", err.Error()), nil)
	}
	return configs, nil
}

// GetStorageConfig gets a specific storage configuration by ID
func GetStorageConfig(storageID string) (*StorageConfig, error) {
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/storage/id/%s", storageID), nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var config StorageConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal storage config: %s", err.Error()), nil)
	}
	return &config, nil
}

// CreateStorageConfig creates a new storage configuration
func CreateStorageConfig(req CreateStorageRequest) (*StorageConfig, error) {
	var resp apiResponse
	err := makeRequest("POST", "/storage/create", req, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var config StorageConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal storage config: %s", err.Error()), nil)
	}
	return &config, nil
}

// DeleteStorageConfig deletes a storage configuration
func DeleteStorageConfig(storageID string) error {
	return makeRequest("DELETE", fmt.Sprintf("/storage/%s", storageID), nil, nil)
}

// ListOrganizationBuckets lists all buckets for an organization
func ListOrganizationBuckets(orgID string) ([]S3Bucket, error) {
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/storage/organizations/%s/buckets", orgID), nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var buckets []S3Bucket
	if err := json.Unmarshal(data, &buckets); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal buckets: %s", err.Error()), nil)
	}
	return buckets, nil
}

// CreateOrganizationBucket creates a new bucket for an organization
func CreateOrganizationBucket(orgID, bucketName string) (*S3Bucket, error) {
	req := map[string]string{"bucket_name": bucketName}

	var resp apiResponse
	err := makeRequest("POST", fmt.Sprintf("/storage/organizations/%s/buckets", orgID), req, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var bucket S3Bucket
	if err := json.Unmarshal(data, &bucket); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal bucket: %s", err.Error()), nil)
	}
	return &bucket, nil
}

// DeleteBucket deletes a bucket
func DeleteBucket(bucketName string) error {
	return makeRequest("DELETE", fmt.Sprintf("/storage/buckets/%s", bucketName), nil, nil)
}

// ListFiles lists files in a storage configuration
func ListFiles(storageID string) ([]S3Object, error) {
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/storage/files/%s", storageID), nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var files []S3Object
	if err := json.Unmarshal(data, &files); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal files: %s", err.Error()), nil)
	}
	return files, nil
}

// DeleteFile deletes a file from storage
func DeleteFile(objectID string) error {
	return makeRequest("DELETE", fmt.Sprintf("/storage/files/%s", objectID), nil, nil)
}

// GetPresignedURL gets a presigned URL for file access
func GetPresignedURL(storageID, fileName string, expiresIn int) (*PresignedURLResponse, error) {
	path := fmt.Sprintf("/storage/presigned-url/%s?file=%s", storageID, fileName)
	if expiresIn > 0 {
		path = fmt.Sprintf("%s&expires=%d", path, expiresIn)
	}

	var resp apiResponse
	err := makeRequest("GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var presigned PresignedURLResponse
	if err := json.Unmarshal(data, &presigned); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal presigned URL: %s", err.Error()), nil)
	}
	return &presigned, nil
}

// GetStorageUsage gets usage statistics for a storage configuration
func GetStorageUsage(storageID string) (*StorageUsage, error) {
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/storage/usage/%s", storageID), nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var usage StorageUsage
	if err := json.Unmarshal(data, &usage); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal storage usage: %s", err.Error()), nil)
	}
	return &usage, nil
}

// FormatBytes formats bytes to human-readable format
func FormatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/TB)
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
