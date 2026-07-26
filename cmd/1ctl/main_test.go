package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateCommand(t *testing.T) {
	cmd := createCommand()
	if cmd.Name != "1ctl" {
		t.Errorf("Expected command name '1ctl', got %s", cmd.Name)
	}
}

func TestPackageCommandIsRegistered(t *testing.T) {
	cmd := createCommand()
	for _, subcommand := range cmd.Commands {
		if subcommand.Name == "package" {
			return
		}
	}
	t.Fatal("package command is not registered")
}

func TestPackageCreateIsLocalAndUsesItsOutputFlag(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "satusky.toml")
	outputPath := filepath.Join(dir, "demo.tar.gz")
	config := `[app]
name = "demo"
port = 8080

[build]
image = "registry.example.test/demo@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
`
	if err := os.WriteFile(configPath, []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	if err := createCommand().Run(context.Background(), []string{"1ctl", "package", "create", "--config", configPath, "--output", outputPath}); err != nil {
		t.Fatalf("package create failed without authentication: %v", err)
	}
	if info, err := os.Stat(outputPath); err != nil || info.Size() == 0 {
		t.Fatalf("package output = %v, %v; want non-empty artifact", info, err)
	}
}
