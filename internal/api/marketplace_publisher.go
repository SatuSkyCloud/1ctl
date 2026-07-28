package api

import (
	"1ctl/internal/config"
	cliContext "1ctl/internal/context"
	"1ctl/internal/utils"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// MarketplacePackageArtifact is the publisher-facing projection of an
// immutable package release. Archive bytes are deliberately never exposed.
type MarketplacePackageArtifact struct {
	MarketplaceID       string    `json:"marketplace_id,omitempty"`
	ReleaseID           string    `json:"release_id"`
	OwnerOrganizationID string    `json:"owner_organization_id,omitempty"`
	ArchiveDigest       string    `json:"archive_digest"`
	Visibility          string    `json:"visibility"`
	CreatedAt           time.Time `json:"created_at,omitempty" tstype:",required"`
}

// MarketplacePackageArtifactTombstone confirms a logical release deletion.
// The package archive remains retained by the backend and is never returned.
type MarketplacePackageArtifactTombstone struct {
	ReleaseID string    `json:"release_id"`
	DeletedAt time.Time `json:"deleted_at" tstype:",required"`
	DeletedBy string    `json:"deleted_by"`
}

func UploadMarketplacePackageArtifact(organizationID, packageName string, archive []byte) (*MarketplacePackageArtifact, error) {
	if len(archive) == 0 {
		return nil, fmt.Errorf("package archive is empty")
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("package_name", packageName); err != nil {
		return nil, fmt.Errorf("write package name: %w", err)
	}
	part, err := writer.CreateFormFile("archive", packageName+".tar.gz")
	if err != nil {
		return nil, fmt.Errorf("create archive form field: %w", err)
	}
	if _, err = part.Write(archive); err != nil {
		return nil, fmt.Errorf("write archive form field: %w", err)
	}
	if err = writer.Close(); err != nil {
		return nil, fmt.Errorf("finish package upload: %w", err)
	}
	var response struct {
		Error bool                       `json:"error"`
		Data  MarketplacePackageArtifact `json:"data"`
	}
	path := fmt.Sprintf("/organizations/%s/releases", url.PathEscape(organizationID))
	if err := makePublisherRequest(http.MethodPost, path, &body, writer.FormDataContentType(), &response); err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func ListMarketplacePackageArtifacts(organizationID string) ([]MarketplacePackageArtifact, error) {
	var response struct {
		Error bool                         `json:"error"`
		Data  []MarketplacePackageArtifact `json:"data"`
	}
	path := fmt.Sprintf("/organizations/%s/releases", url.PathEscape(organizationID))
	if err := makePublisherJSONRequest(http.MethodGet, path, nil, &response); err != nil {
		return nil, err
	}
	return response.Data, nil
}

func GetMarketplacePackageArtifact(organizationID, releaseID string) (*MarketplacePackageArtifact, error) {
	var response struct {
		Error bool                       `json:"error"`
		Data  MarketplacePackageArtifact `json:"data"`
	}
	path := fmt.Sprintf("/organizations/%s/releases/%s", url.PathEscape(organizationID), url.PathEscape(releaseID))
	if err := makePublisherJSONRequest(http.MethodGet, path, nil, &response); err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func RequestMarketplacePackageArtifactPublic(organizationID, releaseID, reason string) (*MarketplacePackageArtifact, error) {
	var response struct {
		Error bool                       `json:"error"`
		Data  MarketplacePackageArtifact `json:"data"`
	}
	path := fmt.Sprintf("/organizations/%s/releases/%s/request-public", url.PathEscape(organizationID), url.PathEscape(releaseID))
	if err := makePublisherJSONRequest(http.MethodPost, path, map[string]string{"reason": reason}, &response); err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func DeleteMarketplacePackageArtifact(organizationID, releaseID string) (*MarketplacePackageArtifactTombstone, error) {
	var response struct {
		Error bool                                `json:"error"`
		Data  MarketplacePackageArtifactTombstone `json:"data"`
	}
	path := fmt.Sprintf("/organizations/%s/releases/%s", url.PathEscape(organizationID), url.PathEscape(releaseID))
	if err := makePublisherJSONRequest(http.MethodDelete, path, nil, &response); err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func makePublisherJSONRequest(method, path string, body any, response any) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal publisher request: %w", err)
		}
		reader = bytes.NewReader(data)
	}
	return makePublisherRequest(method, path, reader, "application/json", response)
}

func makePublisherRequest(method, requestPath string, body io.Reader, contentType string, response any) error {
	requestURL := publisherAPIURL(requestPath)
	if !utils.IsLocalhostURL(requestURL) && !strings.HasPrefix(requestURL, "https://") {
		return utils.NewError(fmt.Sprintf("refusing to send auth token over insecure connection (%s). Use HTTPS or http://localhost for local development", requestURL), nil)
	}
	token := cliContext.GetToken()
	if token == "" {
		return utils.NewError("not authenticated. Please run '1ctl auth login' to authenticate", nil)
	}
	req, err := http.NewRequest(method, requestURL, body)
	if err != nil {
		return fmt.Errorf("create publisher request: %w", err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("x-satusky-api-key", token)
	if email := cliContext.GetEmail(); email != "" {
		req.Header.Set("x-satusky-user-email", email)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to make request: %s", err.Error()), nil)
	}
	defer func() {
		_ = resp.Body.Close() //nolint:errcheck // Response processing has already completed.
	}()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read publisher response: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return newHTTPStatusError(resp.StatusCode, data, parseRetryAfter(resp.Header.Get("Retry-After")))
	}
	if response != nil && len(data) > 0 {
		if err := json.Unmarshal(data, response); err != nil {
			return fmt.Errorf("parse publisher response: %w", err)
		}
	}
	return nil
}

func publisherAPIURL(requestPath string) string {
	base := strings.TrimSuffix(config.GetConfig().ApiURL, "/")
	base = strings.TrimSuffix(base, "/cli")
	base = strings.TrimSuffix(base, "/")
	return base + "/marketplace-publisher" + requestPath
}
