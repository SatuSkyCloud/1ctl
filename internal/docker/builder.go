package docker

import (
	"1ctl/internal/utils"
	"1ctl/internal/validator"
	"fmt"
	"path/filepath"
)

// DockerClient interface for mocking in tests
type DockerClient interface {
	Build(opts BuildOptions) error
}

// Builder handles Docker image building operations
type Builder struct {
	dockerfilePath string
	projectName    string
	context        string
	client         DockerClient
}

// NewBuilder creates a new Docker builder instance
func NewBuilder(dockerfilePath, projectName string) *Builder {
	return &Builder{
		dockerfilePath: dockerfilePath,
		projectName:    projectName,
		context:        filepath.Dir(dockerfilePath),
		client:         &defaultDockerClient{},
	}
}

// defaultDockerClient implements real Docker operations
type defaultDockerClient struct{}

func (d *defaultDockerClient) Build(opts BuildOptions) error {
	return Build(opts)
}

// Build builds a Docker image using the specified Dockerfile
func (b *Builder) Build() (string, error) {
	// Validate Dockerfile
	if err := validator.ValidateDockerfile(b.dockerfilePath); err != nil {
		return "", utils.NewError(fmt.Sprintf("invalid Dockerfile: %s", err.Error()), nil)
	}

	// Build the image
	opts := BuildOptions{
		DockerfilePath: b.dockerfilePath,
		Tag:            b.projectName,
		Context:        b.context,
	}

	if err := b.client.Build(opts); err != nil {
		return "", utils.NewError(fmt.Sprintf("failed to build image: %s", err.Error()), nil)
	}

	return b.projectName, nil
}

// SetContext sets a custom build context directory
func (b *Builder) SetContext(context string) {
	b.context = context
}
