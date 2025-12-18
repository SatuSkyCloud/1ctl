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
	// Multi-cluster deployment options
	MulticlusterEnabled   bool
	MulticlusterMode      string // "active-active" or "active-passive"
	BackupEnabled         bool   // Enable backups (auto-enabled for active-passive, optional for active-active)
	BackupSchedule        string // "hourly", "daily", "weekly"
	BackupRetention       string // "24h", "72h", "168h", "720h"
	BackupPriorityCluster int    // Which cluster performs backups (1 = primary, 2 = secondary)
}
