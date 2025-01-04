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
			errCheck: func(err error) bool { return err.Error() == "dockerfile is empty" },
		},
		{
			name: "missing FROM",
			content: `WORKDIR /app
COPY . .`,
			wantErr: true,
			errCheck: func(err error) bool {
				return strings.Contains(err.Error(), "First instruction must be FROM")
			},
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
