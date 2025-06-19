package validator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func createTempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "validator-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func createTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	return path
}

func TestValidateDockerfile(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantErr  bool
		errCheck func(error) bool
	}{
		{
			name: "valid dockerfile",
			content: `FROM golang:1.21
WORKDIR /app
COPY . .
RUN go build -o main .
CMD ["./main"]`,
			wantErr: false,
		},
		{
			name:     "empty dockerfile",
			content:  "",
			wantErr:  true,
			errCheck: func(err error) bool { return strings.Contains(err.Error(), "dockerfile is empty") },
		},
		{
			name: "missing FROM",
			content: `WORKDIR /app
COPY . .`,
			wantErr: true,
			errCheck: func(err error) bool {
				return strings.Contains(err.Error(), "dockerfile validation failed")
			},
		},
		{
			name: "multistage dockerfile",
			content: `FROM golang:1.21-alpine as builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download && go mod tidy
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/app/main.go

FROM alpine:3.6
WORKDIR /root/
COPY --from=builder /app/main .
EXPOSE 8080
CMD ["./main"]`,
			wantErr: false,
		},
		{
			name: "multistage dockerfile with line continuations",
			content: `FROM golang:1.23-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -ldflags="-s -w" -o apiserver cmd/app/main.go

FROM alpine:latest
COPY --from=builder /build/apiserver /
RUN apk add --no-cache \
    curl \
    openssl \
    git \
    && curl -LO https://example.com/kubectl \
    && chmod +x ./kubectl \
    && mv ./kubectl /usr/local/bin
EXPOSE 8080
ENTRYPOINT ["/apiserver"]`,
			wantErr: false,
		},
		{
			name: "invalid stage name with special characters",
			content: `FROM golang:1.21-alpine AS builder@stage
WORKDIR /app
FROM alpine:3.6
COPY --from=builder@stage /app/main .`,
			wantErr: true,
			errCheck: func(err error) bool {
				return strings.Contains(err.Error(), "dockerfile validation failed")
			},
		},
		{
			name: "invalid multistage FROM syntax",
			content: `FROM golang:1.21-alpine AS
WORKDIR /app`,
			wantErr: true,
			errCheck: func(err error) bool {
				return strings.Contains(err.Error(), "dockerfile validation failed")
			},
		},
		{
			name: "valid COPY --from with stage name",
			content: `FROM golang:1.21-alpine AS build-stage
WORKDIR /app
COPY . .
RUN go build -o app main.go

FROM alpine:latest
COPY --from=build-stage /app/app /usr/local/bin/
CMD ["/usr/local/bin/app"]`,
			wantErr: false,
		},
		{
			name: "valid image names with various formats",
			content: `FROM registry.example.com:5000/my-org/golang:1.21-alpine AS builder
WORKDIR /app

FROM alpine:latest
COPY --from=builder /app/binary /
CMD ["/binary"]`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "Dockerfile")
			if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			err := ValidateDockerfile(path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDockerfile() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.errCheck != nil && err != nil && !tt.errCheck(err) {
				t.Errorf("ValidateDockerfile() error = %v, did not match expected error", err)
			}
		})
	}
}

func TestFindDockerfile(t *testing.T) {
	tests := []struct {
		name         string
		files        map[string]string
		expectedPath string
		expectErr    bool
	}{
		{
			name: "root dockerfile",
			files: map[string]string{
				"Dockerfile": "FROM golang:1.21\nWORKDIR /app",
			},
			expectedPath: "Dockerfile",
			expectErr:    false,
		},
		{
			name: "docker directory",
			files: map[string]string{
				"docker/Dockerfile": "FROM golang:1.21\nWORKDIR /app",
			},
			expectedPath: "docker/Dockerfile",
			expectErr:    false,
		},
		{
			name:      "no dockerfile",
			files:     map[string]string{},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := createTempDir(t)

			for path, content := range tt.files {
				fullPath := filepath.Join(dir, path)
				err := os.MkdirAll(filepath.Dir(fullPath), 0755)
				if err != nil {
					t.Fatalf("failed to create directory: %v", err)
				}
				createTestFile(t, filepath.Dir(fullPath), filepath.Base(path), content)
			}

			path, err := FindDockerfile(dir)
			if (err != nil) != tt.expectErr {
				t.Errorf("FindDockerfile() error = %v, expectErr %v", err, tt.expectErr)
			}
			if !tt.expectErr && !strings.HasSuffix(path, tt.expectedPath) {
				t.Errorf("FindDockerfile() = %v, want %v", path, tt.expectedPath)
			}
		})
	}
}
