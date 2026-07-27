package packagecmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"1ctl/internal/api"
	"1ctl/internal/config"
	satuskyctx "1ctl/internal/context"
	"1ctl/internal/packageartifact"
	"1ctl/internal/utils"

	"github.com/google/uuid"
)

func handleCreate(_ context.Context, input createInput) error {
	var (
		archive     []byte
		packageName string
		err         error
	)
	if input.Chart != "" {
		if input.Config != "" || input.Image != "" {
			return fmt.Errorf("--chart cannot be combined with --config or --image")
		}
		if input.Output == "" {
			return fmt.Errorf("--output is required with --chart")
		}
		archive, packageName, err = packageartifact.CreateHelm(input.Chart)
	} else {
		project, loadErr := config.LoadConfig(input.Config)
		if loadErr != nil {
			return fmt.Errorf("load package config: %w", loadErr)
		}
		archive, packageName, err = packageartifact.Create(project, input.Image)
	}
	if err != nil {
		return err
	}
	output := input.Output
	if output == "" {
		output = packageName + ".tar.gz"
	}
	if filepath.Ext(output) != ".gz" || !strings.HasSuffix(output, ".tar.gz") {
		return fmt.Errorf("package output must end in .tar.gz")
	}
	// output is an explicit user-selected local artifact path and O_EXCL
	// prevents overwriting an existing file.
	file, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600) // #nosec G304
	if err != nil {
		return fmt.Errorf("create package artifact %s: %w", output, err)
	}
	_, writeErr := file.Write(archive)
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		return fmt.Errorf("write package artifact %s", output)
	}
	if utils.TryPrintJSON(map[string]string{"package_name": packageName, "output": output}) {
		return nil
	}
	utils.PrintSuccess("Created unsigned marketplace package: %s", output)
	return nil
}

func handlePublish(_ context.Context, input publishInput) error {
	organizationID, err := currentOrganizationID()
	if err != nil {
		return err
	}
	artifactPath := input.Artifact
	if artifactPath == "" {
		return fmt.Errorf("package artifact is required")
	}
	// artifactPath is an explicit CLI input; ArchivePackageName validates the
	// archive before it is uploaded.
	archive, err := os.ReadFile(artifactPath) // #nosec G304
	if err != nil {
		return fmt.Errorf("read package artifact %s: %w", artifactPath, err)
	}
	if !strings.HasSuffix(artifactPath, ".tar.gz") {
		return fmt.Errorf("package artifact must end in .tar.gz")
	}
	packageName, err := packageartifact.ArchivePackageName(archive)
	if err != nil {
		return fmt.Errorf("read package artifact name: %w", err)
	}
	release, err := api.UploadMarketplacePackageArtifact(organizationID, packageName, archive)
	if err != nil {
		return fmt.Errorf("publish package: %w", err)
	}
	if input.Public {
		release, err = api.RequestMarketplacePackageArtifactPublic(organizationID, release.ReleaseID, input.Reason)
		if err != nil {
			return fmt.Errorf("request public package review: %w", err)
		}
	}
	printArtifact(release)
	return nil
}

func handleList(_ context.Context) error {
	organizationID, err := currentOrganizationID()
	if err != nil {
		return err
	}
	artifacts, err := api.ListMarketplacePackageArtifacts(organizationID)
	if err != nil {
		return fmt.Errorf("list published packages: %w", err)
	}
	if utils.PrintListOrJSON(artifacts, "No marketplace package releases published by the active organization") {
		return nil
	}
	rows := make([][]string, 0, len(artifacts))
	for _, artifact := range artifacts {
		rows = append(rows, []string{artifact.MarketplaceID, artifact.ReleaseID, artifact.ArchiveDigest, artifact.Visibility})
	}
	utils.PrintTable([]string{"MARKETPLACE ID", "RELEASE ID", "ARCHIVE DIGEST", "VISIBILITY"}, rows)
	return nil
}

func handleStatus(_ context.Context, releaseID string) error {
	organizationID, err := currentOrganizationID()
	if err != nil {
		return err
	}
	artifact, err := api.GetMarketplacePackageArtifact(organizationID, releaseID)
	if err != nil {
		return fmt.Errorf("get package status: %w", err)
	}
	if utils.TryPrintJSON(artifact) {
		return nil
	}
	printArtifact(artifact)
	return nil
}

func currentOrganizationID() (string, error) {
	organizationID := strings.TrimSpace(satuskyctx.GetCurrentOrgID())
	if organizationID == "" {
		return "", fmt.Errorf("active organization ID is missing; run '1ctl auth login' or '1ctl org switch <name>'")
	}
	if _, err := uuid.Parse(organizationID); err != nil {
		return "", fmt.Errorf("active organization ID is invalid; run '1ctl auth login' again")
	}
	return organizationID, nil
}

func printArtifact(artifact *api.MarketplacePackageArtifact) {
	if utils.TryPrintJSON(artifact) {
		return
	}
	utils.PrintStatusLine("Marketplace ID", artifact.MarketplaceID)
	utils.PrintStatusLine("Release ID", artifact.ReleaseID)
	utils.PrintStatusLine("Archive Digest", artifact.ArchiveDigest)
	utils.PrintStatusLine("Visibility", artifact.Visibility)
}
