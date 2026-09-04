package cliapp

import (
	"context"
	"strings"
	"testing"
)

func TestValidateOutputFormat(t *testing.T) {
	for _, format := range []string{"table", "wide", "json"} {
		if err := validateOutputFormat(format); err != nil {
			t.Errorf("validateOutputFormat(%q) error = %v", format, err)
		}
	}

	err := validateOutputFormat("yaml")
	if err == nil || !strings.Contains(err.Error(), "expected table, wide or json") {
		t.Fatalf("validateOutputFormat(\"yaml\") error = %v, want supported formats", err)
	}
}

func TestRootRejectsUnsupportedOutputBeforeCommandAction(t *testing.T) {
	err := New().Run(context.Background(), []string{
		"1ctl", "--output", "yaml", "profile", "list",
	})
	if err == nil {
		t.Fatal("Run() error = nil, want unsupported output error")
	}
	if !strings.Contains(err.Error(), `invalid --output "yaml"`) {
		t.Fatalf("Run() error = %q, want unsupported output error", err)
	}
}
