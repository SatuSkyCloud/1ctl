package commands

import (
	"1ctl/internal/api"
	"1ctl/internal/context"
	"1ctl/internal/utils"
	"fmt"

	"github.com/urfave/cli/v2"
)

func StorageCommand() *cli.Command {
	return &cli.Command{
		Name:    "storage",
		Aliases: []string{"s3", "spaces"},
		Usage:   "Manage S3/object storage",
		Subcommands: []*cli.Command{
			storageListCommand(),
			storageGetCommand(),
			storageBucketsCommand(),
			storageFilesCommand(),
			storageUsageCommand(),
			storagePresignCommand(),
			storageDeleteCommand(),
		},
	}
}

func storageListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List storage configurations",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "namespace",
				Usage: "Namespace to list storage from (default: current namespace)",
			},
		},
		Action: handleStorageList,
	}
}

func storageGetCommand() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "Get storage configuration details",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "storage-id",
				Usage:    "Storage ID to retrieve",
				Required: true,
			},
		},
		Action: handleStorageGet,
	}
}

func storageBucketsCommand() *cli.Command {
	return &cli.Command{
		Name:  "buckets",
		Usage: "Manage S3 buckets",
		Subcommands: []*cli.Command{
			{
				Name:   "list",
				Usage:  "List organization buckets",
				Action: handleBucketsList,
			},
			{
				Name:  "create",
				Usage: "Create a new bucket",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "name",
						Usage:    "Bucket name",
						Required: true,
					},
				},
				Action: handleBucketCreate,
			},
			{
				Name:  "delete",
				Usage: "Delete a bucket",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "name",
						Usage:    "Bucket name to delete",
						Required: true,
					},
				},
				Action: handleBucketDelete,
			},
		},
		Action: handleBucketsList, // Default action shows list
	}
}

func storageFilesCommand() *cli.Command {
	return &cli.Command{
		Name:  "files",
		Usage: "List files in storage",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "storage-id",
				Usage:    "Storage ID to list files from",
				Required: true,
			},
		},
		Action: handleFilesList,
	}
}

func storageUsageCommand() *cli.Command {
	return &cli.Command{
		Name:  "usage",
		Usage: "Show storage usage",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "storage-id",
				Usage:    "Storage ID to show usage for",
				Required: true,
			},
		},
		Action: handleStorageUsage,
	}
}

func storagePresignCommand() *cli.Command {
	return &cli.Command{
		Name:  "presign",
		Usage: "Get presigned URL for file access",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "storage-id",
				Usage:    "Storage ID",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "file",
				Usage:    "File name/key",
				Required: true,
			},
			&cli.IntFlag{
				Name:  "expires",
				Usage: "Expiration time in seconds (default: 3600)",
				Value: 3600,
			},
		},
		Action: handleStoragePresign,
	}
}

func storageDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete a storage configuration",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "storage-id",
				Usage:    "Storage ID to delete",
				Required: true,
			},
		},
		Action: handleStorageDelete,
	}
}

func handleStorageList(c *cli.Context) error {
	namespace := c.String("namespace")
	if namespace == "" {
		namespace = context.GetCurrentNamespace()
	}

	if namespace == "" {
		return utils.NewError("namespace not found. Please set a namespace or login first", nil)
	}

	configs, err := api.GetStorageConfigs(namespace)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list storage: %s", err.Error()), nil)
	}

	if len(configs) == 0 {
		utils.PrintInfo("No storage configurations found in namespace %s", namespace)
		return nil
	}

	utils.PrintHeader("Storage Configurations")
	for _, cfg := range configs {
		usedPercent := 0.0
		if cfg.SizeBytes > 0 {
			usedPercent = float64(cfg.UsedBytes) / float64(cfg.SizeBytes) * 100
		}

		utils.PrintStatusLine("ID", cfg.StorageID.String())
		utils.PrintStatusLine("Name", cfg.Name)
		utils.PrintStatusLine("Bucket", cfg.BucketName)
		utils.PrintStatusLine("Type", cfg.Type)
		utils.PrintStatusLine("Size", api.FormatBytes(cfg.SizeBytes))
		utils.PrintStatusLine("Used", fmt.Sprintf("%s (%.1f%%)", api.FormatBytes(cfg.UsedBytes), usedPercent))
		utils.PrintStatusLine("Created", formatTimeAgo(cfg.CreatedAt))
		utils.PrintDivider()
	}
	return nil
}

func handleStorageGet(c *cli.Context) error {
	storageID := c.String("storage-id")
	if storageID == "" {
		return utils.NewError("--storage-id is required", nil)
	}

	config, err := api.GetStorageConfig(storageID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get storage: %s", err.Error()), nil)
	}

	usedPercent := 0.0
	if config.SizeBytes > 0 {
		usedPercent = float64(config.UsedBytes) / float64(config.SizeBytes) * 100
	}

	utils.PrintHeader("Storage Details")
	utils.PrintStatusLine("ID", config.StorageID.String())
	utils.PrintStatusLine("Name", config.Name)
	utils.PrintStatusLine("Bucket", config.BucketName)
	utils.PrintStatusLine("Type", config.Type)
	utils.PrintStatusLine("Size", api.FormatBytes(config.SizeBytes))
	utils.PrintStatusLine("Used", fmt.Sprintf("%s (%.1f%%)", api.FormatBytes(config.UsedBytes), usedPercent))
	utils.PrintStatusLine("Endpoint", config.Endpoint)
	utils.PrintStatusLine("Namespace", config.Namespace)
	utils.PrintStatusLine("Created", formatTimeAgo(config.CreatedAt))
	utils.PrintStatusLine("Updated", formatTimeAgo(config.UpdatedAt))
	return nil
}

func handleBucketsList(c *cli.Context) error {
	orgID := context.GetCurrentOrgID()
	if orgID == "" {
		return utils.NewError("organization ID not found. Please run '1ctl auth login' first", nil)
	}

	buckets, err := api.ListOrganizationBuckets(orgID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list buckets: %s", err.Error()), nil)
	}

	if len(buckets) == 0 {
		utils.PrintInfo("No buckets found")
		return nil
	}

	utils.PrintHeader("Organization Buckets")
	for _, bucket := range buckets {
		utils.PrintStatusLine("Name", bucket.BucketName)
		utils.PrintStatusLine("Objects", fmt.Sprintf("%d", bucket.ObjectCount))
		utils.PrintStatusLine("Size", api.FormatBytes(bucket.SizeBytes))
		utils.PrintStatusLine("Created", formatTimeAgo(bucket.CreatedAt))
		utils.PrintDivider()
	}
	return nil
}

func handleBucketCreate(c *cli.Context) error {
	orgID := context.GetCurrentOrgID()
	if orgID == "" {
		return utils.NewError("organization ID not found. Please run '1ctl auth login' first", nil)
	}

	name := c.String("name")
	if name == "" {
		return utils.NewError("--name is required", nil)
	}

	bucket, err := api.CreateOrganizationBucket(orgID, name)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to create bucket: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Bucket created successfully!")
	utils.PrintStatusLine("Name", bucket.BucketName)
	utils.PrintStatusLine("Created", formatTimeAgo(bucket.CreatedAt))
	return nil
}

func handleBucketDelete(c *cli.Context) error {
	name := c.String("name")
	if name == "" {
		return utils.NewError("--name is required", nil)
	}

	if err := api.DeleteBucket(name); err != nil {
		return utils.NewError(fmt.Sprintf("failed to delete bucket: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Bucket '%s' deleted successfully", name)
	return nil
}

func handleFilesList(c *cli.Context) error {
	storageID := c.String("storage-id")
	if storageID == "" {
		return utils.NewError("--storage-id is required", nil)
	}

	files, err := api.ListFiles(storageID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list files: %s", err.Error()), nil)
	}

	if len(files) == 0 {
		utils.PrintInfo("No files found in storage")
		return nil
	}

	utils.PrintHeader("Files")
	var totalSize int64
	for _, file := range files {
		utils.PrintStatusLine("Key", file.Key)
		utils.PrintStatusLine("Size", api.FormatBytes(file.Size))
		utils.PrintStatusLine("Type", file.ContentType)
		utils.PrintStatusLine("Modified", formatTimeAgo(file.LastModified))
		utils.PrintDivider()
		totalSize += file.Size
	}
	fmt.Printf("\nTotal: %d files, %s\n", len(files), api.FormatBytes(totalSize))
	return nil
}

func handleStorageUsage(c *cli.Context) error {
	storageID := c.String("storage-id")
	if storageID == "" {
		return utils.NewError("--storage-id is required", nil)
	}

	usage, err := api.GetStorageUsage(storageID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get storage usage: %s", err.Error()), nil)
	}

	usedPercent := 0.0
	availableBytes := usage.TotalBytes - usage.UsedBytes
	if usage.TotalBytes > 0 {
		usedPercent = float64(usage.UsedBytes) / float64(usage.TotalBytes) * 100
	}

	// Create progress bar
	barWidth := 20
	filledWidth := int(usedPercent / 100 * float64(barWidth))
	progressBar := ""
	for i := 0; i < barWidth; i++ {
		if i < filledWidth {
			progressBar += "█"
		} else {
			progressBar += "░"
		}
	}

	utils.PrintHeader("Storage Usage")
	utils.PrintStatusLine("Storage ID", usage.StorageID.String())
	utils.PrintStatusLine("Total Size", api.FormatBytes(usage.TotalBytes))
	utils.PrintStatusLine("Used", api.FormatBytes(usage.UsedBytes))
	utils.PrintStatusLine("Available", api.FormatBytes(availableBytes))
	utils.PrintStatusLine("Objects", fmt.Sprintf("%d", usage.ObjectCount))
	utils.PrintStatusLine("Usage", fmt.Sprintf("%s %.1f%%", progressBar, usedPercent))
	return nil
}

func handleStoragePresign(c *cli.Context) error {
	storageID := c.String("storage-id")
	fileName := c.String("file")
	expires := c.Int("expires")

	if storageID == "" {
		return utils.NewError("--storage-id is required", nil)
	}
	if fileName == "" {
		return utils.NewError("--file is required", nil)
	}

	presigned, err := api.GetPresignedURL(storageID, fileName, expires)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get presigned URL: %s", err.Error()), nil)
	}

	expiresIn := "1 hour"
	if expires != 3600 {
		expiresIn = fmt.Sprintf("%d seconds", expires)
	}

	utils.PrintHeader("Presigned URL")
	utils.PrintStatusLine("File", fileName)
	utils.PrintStatusLine("Expires", expiresIn)
	utils.PrintStatusLine("URL", presigned.URL)
	return nil
}

func handleStorageDelete(c *cli.Context) error {
	storageID := c.String("storage-id")
	if storageID == "" {
		return utils.NewError("--storage-id is required", nil)
	}

	if err := api.DeleteStorageConfig(storageID); err != nil {
		return utils.NewError(fmt.Sprintf("failed to delete storage: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Storage '%s' deleted successfully", storageID)
	return nil
}
