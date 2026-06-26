package init

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"1ctl/internal/config"
	"1ctl/internal/utils"
)

func handleInit(ctx context.Context, in initInput) error {
	configArg := in.Config
	filename := config.DefaultConfigFile
	if configArg != "" && !strings.HasSuffix(configArg, ".toml") {
		filename = fmt.Sprintf("satusky.%s.toml", configArg)
	} else if configArg != "" {
		filename = configArg
	}

	if _, err := os.Stat(filename); err == nil {
		return utils.NewError(fmt.Sprintf("%s already exists", filename), nil)
	}

	var base config.ProjectConfig
	if filename != config.DefaultConfigFile {
		if existing, err := config.FindConfig(""); err == nil && existing != nil {
			base = *existing
		}
	}

	dir, err := os.Getwd()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get working directory: %s", err.Error()), nil)
	}
	if base.App.Name == "" {
		base.App.Name = filepath.Base(dir)
	}
	if base.Build.Dockerfile == "" {
		base.Build.Dockerfile = "Dockerfile"
	}
	if base.App.Port == 0 {
		base.App.Port = 8080
	}

	base.Path = filename
	lines := []string{"[app]"}
	lines = append(lines, fmt.Sprintf("  name = %q", base.App.Name))
	if base.App.Port != 0 {
		lines = append(lines, fmt.Sprintf("  port = %d", base.App.Port))
	}
	if base.App.CPURequest != "" {
		lines = append(lines, fmt.Sprintf("  cpu_request = %q", base.App.CPURequest))
	}
	if base.App.CPULimit != "" {
		lines = append(lines, fmt.Sprintf("  cpu_limit = %q", base.App.CPULimit))
	} else if base.App.CPU != "" {
		lines = append(lines, fmt.Sprintf("  cpu = %q", base.App.CPU))
	}
	if base.App.Memory != "" {
		lines = append(lines, fmt.Sprintf("  memory = %q", base.App.Memory))
	}
	if base.App.Replicas > 0 && base.App.Replicas != 1 {
		lines = append(lines, fmt.Sprintf("  replicas = %d", base.App.Replicas))
	}
	if base.App.Domain != "" {
		lines = append(lines, fmt.Sprintf("  domain = %q", base.App.Domain))
	}

	// Commented examples for the [app] section.
	lines = append(lines, "",
		"  # cpu_request = \"250m\"   # guaranteed scheduler reservation",
		"  # cpu_limit = \"1\"        # burst ceiling",
		"  # memory = \"256Mi\"       # platform default 256Mi")

	// [build] section
	lines = append(lines, "", "[build]")
	if base.Build.Dockerfile != "" && base.Build.Dockerfile != "Dockerfile" {
		lines = append(lines, fmt.Sprintf("  dockerfile = %q", base.Build.Dockerfile))
	} else {
		lines = append(lines, "  # dockerfile = \"Dockerfile\"")
	}
	if base.Build.FastBuild {
		lines = append(lines, "  fast_build = true")
	} else {
		lines = append(lines, "  # fast_build = false       # opt into accelerated cloud builds")
	}

	// [checks] section
	lines = append(lines, "", "[checks]")
	if base.Checks.HealthPath != "" {
		lines = append(lines, fmt.Sprintf("  health_path = %q", base.Checks.HealthPath))
	} else {
		lines = append(lines, "  # health_path = \"/health\"  # app-specific HTTP smoke path (used by deploy --wait)")
	}

	// [deploy] section
	lines = append(lines, "", "[deploy]")
	if base.Deploy.Strategy != "" && base.Deploy.Strategy != "rolling" {
		lines = append(lines, fmt.Sprintf("  strategy = %q", base.Deploy.Strategy))
	} else {
		lines = append(lines, "  # strategy = \"rolling\"    # rolling | recreate")
	}
	if base.Deploy.RollingMaxSurge != "" && base.Deploy.RollingMaxSurge != "25%" {
		lines = append(lines, fmt.Sprintf("  rolling_max_surge = %q", base.Deploy.RollingMaxSurge))
	} else {
		lines = append(lines, "  # rolling_max_surge = \"25%\"")
	}
	if base.Deploy.RollingMaxUnavailable != "" && base.Deploy.RollingMaxUnavailable != "25%" {
		lines = append(lines, fmt.Sprintf("  rolling_max_unavailable = %q", base.Deploy.RollingMaxUnavailable))
	} else {
		lines = append(lines, "  # rolling_max_unavailable = \"25%\"")
	}
	if base.Deploy.MachineTag != "" {
		lines = append(lines, fmt.Sprintf("  machine_tag = %q", base.Deploy.MachineTag))
	} else {
		lines = append(lines, "  # machine_tag = \"production\"")
	}
	if len(base.Deploy.WaitFor) > 0 {
		lines = append(lines, "  wait_for = ["+strings.Join(quoteEach(base.Deploy.WaitFor), ", ")+"]")
	} else {
		lines = append(lines, "  # wait_for = [\"postgres:5432\"]")
	}

	// Other optional sections (commented out).
	lines = append(lines, "",
		"# [volume]",
		"#   size = \"10Gi\"",
		"#   mount = \"/data\"",
		"",
		"# [hpa]",
		"#   enabled = true",
		"#   min_replicas = 2",
		"#   max_replicas = 10",
		"#   cpu_target = 80",
		"#   memory_target = 0",
		"",
		"# [vpa]",
		"#   enabled = false",
		"#   mode = \"Off\"  # Off | Initial | Auto",
		"",
		"# [pdb]",
		"#   enabled = false",
		"#   type = \"auto\"  # auto | fixed | percent",
		"#   min_available = 1",
		"#   percent = 50",
		"",
		"# [multicluster]",
		"#   enabled = false",
		"#   mode = \"active-passive\"  # active-active | active-passive",
		"#   backup_enabled = true",
		"#   backup_schedule = \"daily\"",
		"#   backup_retention = \"168h\"",
		"#   backup_priority_cluster = 1",
	)
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(filename, []byte(content), 0600); err != nil {
		return utils.NewError(fmt.Sprintf("failed to write %s: %s", filename, err.Error()), nil)
	}

	utils.PrintSuccess("Created %s", filename)
	if filename != config.DefaultConfigFile {
		utils.PrintInfo("Edit %s to configure resources and domain for this target.", filename)
		if configArg != "" {
			utils.PrintInfo("Then run: 1ctl deploy --config %s", configArg)
		}
	} else {
		utils.PrintInfo("Edit satusky.toml, then run: 1ctl deploy")
	}
	return nil
}

// quoteEach wraps each string in double quotes.
func quoteEach(vals []string) []string {
	out := make([]string, len(vals))
	for i, v := range vals {
		out[i] = fmt.Sprintf("%q", v)
	}
	return out
}
