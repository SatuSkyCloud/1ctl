package api

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type backendClientEndpointManifest struct {
	SchemaVersion int                     `json:"schema_version"`
	BasePath      string                  `json:"base_path"`
	Endpoints     []backendClientEndpoint `json:"endpoints"`
}

type backendClientEndpoint struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

var activeClientEndpoints = []backendClientEndpoint{
	{Method: "POST", Path: "/v1/cli/deployments/{deploymentId}/scale"},
	{Method: "GET", Path: "/v1/marketplaces/all"},
	{Method: "GET", Path: "/v1/marketplaces/id/{marketplaceId}"},
	{Method: "POST", Path: "/v1/marketplaces/deploy/create/{namespace}/{marketplaceId}"},
}

func TestActiveClientEndpointsMatchBackendManifest(t *testing.T) {
	manifestPath, configured := backendManifestPath(t)
	manifestBytes, err := os.ReadFile(manifestPath)
	if errors.Is(err, os.ErrNotExist) && !configured {
		t.Skipf("backend endpoint manifest not found at sibling path %s", manifestPath)
	}
	if err != nil {
		t.Fatalf("read backend endpoint manifest %s: %v", manifestPath, err)
	}

	var manifest backendClientEndpointManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatalf("parse backend endpoint manifest: %v", err)
	}
	if manifest.SchemaVersion != 1 {
		t.Fatalf("backend endpoint manifest schema_version = %d, want 1", manifest.SchemaVersion)
	}

	backendEndpoints := make(map[string]struct{}, len(manifest.Endpoints))
	for _, endpoint := range manifest.Endpoints {
		key := endpoint.Method + " " + endpointManifestPath(manifest.BasePath, endpoint.Path)
		if _, duplicate := backendEndpoints[key]; duplicate {
			t.Fatalf("backend endpoint manifest contains duplicate %s", key)
		}
		backendEndpoints[key] = struct{}{}
	}

	for _, endpoint := range activeClientEndpoints {
		key := endpoint.Method + " " + endpoint.Path
		if _, ok := backendEndpoints[key]; !ok {
			t.Errorf("active 1ctl endpoint %s is absent from backend manifest %s", key, manifestPath)
		}
	}
}

func backendManifestPath(t *testing.T) (string, bool) {
	t.Helper()
	if configured := strings.TrimSpace(os.Getenv("SATUSKY_BACKEND_MANIFEST")); configured != "" {
		return configured, true
	}

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve client endpoint test location")
	}
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	return filepath.Join(repositoryRoot, "..", "satusky-core_backend", "architecture", "client_endpoint_manifest.json"), false
}

func endpointManifestPath(basePath, endpointPath string) string {
	basePath = strings.TrimSuffix(basePath, "/")
	if endpointPath == basePath || strings.HasPrefix(endpointPath, basePath+"/") {
		return endpointPath
	}
	return basePath + endpointPath
}
