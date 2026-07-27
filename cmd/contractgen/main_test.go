package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateAPISourceCoverageRejectsUnlistedWireType(t *testing.T) {
	root := t.TempDir()
	apiDirectory := filepath.Join(root, "internal", "api")
	if err := os.MkdirAll(apiDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	source := "package api\n\ntype NewWireResponse struct { ID string `json:\"id\"` }\n"
	if err := os.WriteFile(filepath.Join(apiDirectory, "new_wire.go"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	err := validateAPISourceCoverage(root, nil)
	if err == nil || !strings.Contains(err.Error(), "new_wire.go:NewWireResponse") {
		t.Fatalf("validateAPISourceCoverage() error = %v, want omitted type", err)
	}
}

func TestValidateAPISourceCoverageAllowsTransportOnlyFiles(t *testing.T) {
	root := t.TempDir()
	apiDirectory := filepath.Join(root, "internal", "api")
	if err := os.MkdirAll(apiDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	source := "package api\n\ntype client struct { endpoint string }\n"
	if err := os.WriteFile(filepath.Join(apiDirectory, "client.go"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := validateAPISourceCoverage(root, nil); err != nil {
		t.Fatal(err)
	}
}
