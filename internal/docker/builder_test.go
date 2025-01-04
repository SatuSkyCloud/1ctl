package docker

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

type mockDockerClient struct {
	buildFunc func(opts BuildOptions) error
}

func (m *mockDockerClient) Build(opts BuildOptions) error {
	if m.buildFunc != nil {
		return m.buildFunc(opts)
	}
	return nil
}

func TestBuildImage(t *testing.T) {
	tests := []struct {
		name              string
		dockerfileContent string
		projectName       string
		mockBuildFunc     func(opts BuildOptions) error
		wantErr           bool
	}{
		{
			name: "valid dockerfile",
			dockerfileContent: `FROM golang:1.21-alpine
WORKDIR /app
COPY . .
CMD ["./app"]`,
			projectName: "test-project",
			mockBuildFunc: func(opts BuildOptions) error {
				return nil
			},
			wantErr: false,
		},
		{
			name:              "invalid dockerfile",
			dockerfileContent: `INVALID`,
			projectName:       "test-project",
			mockBuildFunc: func(opts BuildOptions) error {
				return fmt.Errorf("build failed")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := &mockDockerClient{
				buildFunc: tt.mockBuildFunc,
			}

			// Create test dockerfile
			dir := t.TempDir()
			dockerfilePath := filepath.Join(dir, "Dockerfile")
			if err := os.WriteFile(dockerfilePath, []byte(tt.dockerfileContent), 0644); err != nil {
				t.Fatal(err)
			}

			builder := NewBuilder(dockerfilePath, tt.projectName)
			builder.client = mockClient

			_, err := builder.Build()
			if (err != nil) != tt.wantErr {
				t.Errorf("Build() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBuilderSetContext(t *testing.T) {
	dir := t.TempDir()
	dockerfilePath := filepath.Join(dir, "Dockerfile")
	customContext := t.TempDir()

	builder := NewBuilder(dockerfilePath, "test-image")
	builder.SetContext(customContext)

	if builder.context != customContext {
		t.Errorf("SetContext() = %v, want %v", builder.context, customContext)
	}
}
