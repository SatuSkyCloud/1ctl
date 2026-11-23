package validator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeDockerfile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "Dockerfile")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestFinalStageUser(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{name: "no USER", content: "FROM node:20-alpine\nCMD [\"node\"]\n", want: ""},
		{name: "named USER", content: "FROM node:20-alpine\nUSER node\nCMD [\"node\"]\n", want: "node"},
		{name: "last USER wins", content: "FROM alpine\nUSER root\nRUN apk add curl\nUSER 1000:1000\n", want: "1000:1000"},
		{name: "USER only in a build stage", content: "FROM node:20 AS build\nUSER node\nRUN npm ci\nFROM gcr.io/distroless/nodejs20\nCOPY --from=build /app /app\n", want: ""},
		{name: "final stage inherits from an earlier stage", content: "FROM node:20 AS base\nUSER node\nFROM base\nCMD [\"node\"]\n", want: "node"},
		{name: "inherited user overridden", content: "FROM node:20 AS base\nUSER node\nFROM base\nUSER 1000\n", want: "1000"},
		{name: "--platform and lowercase as", content: "FROM --platform=$BUILDPLATFORM node:20 as base\nUSER node\nFROM base\n", want: "node"},
		{name: "comment mentioning USER", content: "FROM alpine\n# USER nginx\n", want: ""},
		{name: "continuation line", content: "FROM alpine\nUSER \\\n  app\n", want: "app"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FinalStageUser(writeDockerfile(t, tt.content))
			if err != nil {
				t.Fatalf("FinalStageUser() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("FinalStageUser() = %q, want %q", got, tt.want)
			}
		})
	}
}

// Issue #135: USER node under the platform's runAsNonRoot policy never starts,
// and the deployment sat in "reconciling" with no visible reason.
func TestCheckImageUserVerifiable(t *testing.T) {
	uid := int64(1000)
	tests := []struct {
		name      string
		content   string
		runAsUser *int64
		wantErr   []string
	}{
		{
			name:    "named USER from issue #135",
			content: "FROM node:20-alpine\nWORKDIR /app\nUSER node\nCMD [\"node\", \"server.js\"]\n",
			wantErr: []string{"USER node", "run_as_user = 1000", "USER 1000", "uid 1000"},
		},
		{name: "named USER with run_as_user set", content: "FROM node:20-alpine\nUSER node\n", runAsUser: &uid},
		{name: "numeric USER", content: "FROM node:20-alpine\nUSER 1000\n"},
		{name: "numeric USER with group", content: "FROM node:20-alpine\nUSER 1000:1000\n"},
		{name: "no USER", content: "FROM node:20-alpine\n"},
		{name: "USER from a build arg", content: "FROM alpine\nARG APP_USER=app\nUSER $APP_USER\n"},
		{
			name:    "unknown named USER",
			content: "FROM alpine\nUSER appuser\n",
			wantErr: []string{"USER appuser", "run_as_user = <uid>"},
		},
		{name: "USER root", content: "FROM alpine\nUSER root\n", wantErr: []string{"which is root"}},
		{name: "USER 0", content: "FROM alpine\nUSER 0\n", wantErr: []string{"which is root"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckImageUserVerifiable(writeDockerfile(t, tt.content), tt.runAsUser)
			if len(tt.wantErr) == 0 {
				if err != nil {
					t.Fatalf("CheckImageUserVerifiable() error = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatal("CheckImageUserVerifiable() error = nil, want an error")
			}
			for _, want := range tt.wantErr {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error = %q, want it to contain %q", err, want)
				}
			}
		})
	}
}

func TestCheckImageUserVerifiableSkipsWithoutDockerfile(t *testing.T) {
	if err := CheckImageUserVerifiable("", nil); err != nil {
		t.Fatalf("CheckImageUserVerifiable(\"\") error = %v, want nil for prebuilt images", err)
	}
	if err := CheckImageUserVerifiable(filepath.Join(t.TempDir(), "missing"), nil); err != nil {
		t.Fatalf("CheckImageUserVerifiable(missing) error = %v, want nil", err)
	}
}
