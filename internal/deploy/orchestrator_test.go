package deploy

import (
	"testing"

	"1ctl/internal/api"
	"1ctl/internal/context"
	"1ctl/internal/testutils"
)

func TestDeploy(t *testing.T) {
	// Setup test context
	_ = context.SetToken("test-token")
	_ = context.SetUserID("test-user")
	_ = context.SetCurrentNamespace("test-org")

	tests := []struct {
		name    string
		opts    DeploymentOptions
		mockAPI *testutils.MockAPI
		wantErr bool
	}{
		{
			name: "successful deployment",
			opts: DeploymentOptions{
				CPU:            "1",
				Memory:         "512Mi",
				Organization:   "test-project",
				Port:           8080,
				DockerfilePath: "Dockerfile",
			},
			mockAPI: testutils.DefaultMockAPI(),
			wantErr: false,
		},
		{
			name: "deployment with environment",
			opts: DeploymentOptions{
				CPU:            "1",
				Memory:         "512Mi",
				Organization:   "test-project",
				Port:           8080,
				DockerfilePath: "Dockerfile",
				EnvEnabled:     true,
				Environment: &api.Environment{
					KeyValues: []api.KeyValuePair{
						{Key: "TEST_KEY", Value: "test_value"},
					},
				},
			},
			mockAPI: testutils.DefaultMockAPI(),
			wantErr: false,
		},
		{
			name: "deployment with volume",
			opts: DeploymentOptions{
				CPU:            "1",
				Memory:         "512Mi",
				Organization:   "test-project",
				Port:           8080,
				DockerfilePath: "Dockerfile",
				VolumeEnabled:  true,
				Volume: &api.Volume{
					StorageSize: "10Gi",
					MountPath:   "/data",
				},
			},
			mockAPI: testutils.DefaultMockAPI(),
			wantErr: false,
		},
		{
			name: "deployment error",
			opts: DeploymentOptions{
				CPU:            "1",
				Memory:         "512Mi",
				Organization:   "test-project",
				Port:           8080,
				DockerfilePath: "Dockerfile",
			},
			mockAPI: testutils.ErrorMockAPI(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO: Replace actual API calls with mock
			// This requires refactoring the deploy package to accept an API interface

			resp, err := Deploy(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("Deploy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && resp == nil {
				t.Error("Deploy() returned nil response for successful deployment")
			}
		})
	}
}

func TestBuildAndUploadImage(t *testing.T) {
	tests := []struct {
		name           string
		dockerfilePath string
		projectName    string
		wantErr        bool
	}{
		// {
		// 	name:           "valid dockerfile",
		// 	dockerfilePath: "testdata/Dockerfile",
		// 	projectName:    "test-project",
		// 	wantErr:        false,
		// },
		{
			name:           "missing dockerfile",
			dockerfilePath: "testdata/nonexistent",
			projectName:    "test-project",
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := buildAndUploadImage(tt.dockerfilePath, tt.projectName)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildAndUploadImage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
