package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"1ctl/internal/config"
	"1ctl/internal/utils"

	"github.com/urfave/cli/v2"
)

func InitCommand() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "Create a satusky.toml config file in the current directory",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "config",
				Usage: "Config name (e.g. staging → creates satusky.staging.toml)",
			},
		},
		Action: handleInit,
	}
}

func handleInit(c *cli.Context) error {
	configArg := c.String("config")
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
			base.App.DeploymentID = ""
		}
	}

	dir, _ := os.Getwd()
	if base.App.Name == "" {
		base.App.Name = filepath.Base(dir)
	}
	if base.App.Dockerfile == "" {
		base.App.Dockerfile = "Dockerfile"
	}
	if base.App.Port == 0 {
		base.App.Port = 8080
	}

	base.Path = filename
	if err := base.Save(); err != nil {
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
