package deploy

import "1ctl/internal/api"

type DeploymentOptions struct {
	CPU            string
	Memory         string
	Domain         string
	Organization   string
	Port           int
	DockerfilePath string
	Hostnames      []string
	Dependencies   []api.Dependency
	VolumeEnabled  bool
	Volume         *api.Volume
	EnvEnabled     bool
	Environment    *api.Environment
}
