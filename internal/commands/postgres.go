package commands

import (
	"1ctl/internal/api"
	"1ctl/internal/utils"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strings"
	"syscall"

	"github.com/urfave/cli/v2"
)

var postgresNamePattern = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

func PostgresCommand() *cli.Command {
	return &cli.Command{
		Name:    "postgres",
		Aliases: []string{"pg"},
		Usage:   "Manage SatuSky managed Postgres clusters",
		Subcommands: []*cli.Command{
			postgresCreateCommand(),
			postgresListCommand(),
			postgresGetCommand(),
			postgresStatusCommand(),
			postgresCredentialsCommand(),
			postgresConnectCommand(),
			postgresProxyCommand(),
			postgresRedeployCommand(),
			postgresDestroyCommand(),
			postgresUsersCommand(),
			postgresFirewallCommand(),
			postgresStorageClassesCommand(),
		},
	}
}

func postgresCreateCommand() *cli.Command {
	return &cli.Command{
		Name:      "create",
		Usage:     "Create a managed Postgres cluster",
		ArgsUsage: "<name>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "database", Aliases: []string{"d"}, Usage: "Initial database name. Defaults to <name>"},
			&cli.StringFlag{Name: "user", Aliases: []string{"u"}, Usage: "Initial database user", Value: "app"},
			&cli.StringFlag{Name: "version", Usage: "Postgres major version", Value: "17"},
			&cli.IntFlag{Name: "instances", Usage: "CNPG instance count", Value: 1},
			&cli.StringFlag{Name: "storage-size", Usage: "Data volume size", Value: "10Gi"},
			&cli.StringFlag{Name: "storage-class", Usage: "Kubernetes storage class", Required: true},
			&cli.StringFlag{Name: "wal-storage-size", Usage: "WAL volume size. Defaults to data volume size"},
			&cli.StringFlag{Name: "cpu-request", Usage: "CPU request", Value: "250m"},
			&cli.StringFlag{Name: "cpu", Usage: "CPU limit", Value: "1"},
			&cli.StringFlag{Name: "memory-request", Usage: "Memory request", Value: "512Mi"},
			&cli.StringFlag{Name: "memory", Usage: "Memory limit", Value: "1Gi"},
		},
		Action: handlePostgresCreate,
	}
}

func postgresListCommand() *cli.Command {
	return &cli.Command{
		Name:    "list",
		Aliases: []string{"ls"},
		Usage:   "List managed Postgres clusters",
		Action:  handlePostgresList,
	}
}

func postgresGetCommand() *cli.Command {
	return &cli.Command{
		Name:      "get",
		Aliases:   []string{"inspect"},
		Usage:     "Show managed Postgres cluster details",
		ArgsUsage: "<storage-id>",
		Action:    handlePostgresGet,
	}
}

func postgresStatusCommand() *cli.Command {
	return &cli.Command{
		Name:      "status",
		Usage:     "Show live managed Postgres status",
		ArgsUsage: "<storage-id>",
		Action:    handlePostgresStatus,
	}
}

func postgresCredentialsCommand() *cli.Command {
	return &cli.Command{
		Name:      "credentials",
		Aliases:   []string{"creds"},
		Usage:     "Show database connection credentials",
		ArgsUsage: "<storage-id>",
		Action:    handlePostgresCredentials,
	}
}

func postgresConnectCommand() *cli.Command {
	return &cli.Command{
		Name:      "connect",
		Usage:     "Connect to a managed Postgres cluster using psql",
		ArgsUsage: "<storage-id>",
		Action:    handlePostgresConnect,
	}
}

func postgresProxyCommand() *cli.Command {
	return &cli.Command{
		Name:      "proxy",
		Usage:     "Forward a local TCP port to a managed Postgres cluster",
		ArgsUsage: "<storage-id>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "bind-addr", Usage: "Local address to bind", Value: "127.0.0.1"},
			&cli.StringFlag{Name: "local-port", Aliases: []string{"p"}, Usage: "Local port to listen on", Value: "15432"},
		},
		Action: handlePostgresProxy,
	}
}

func postgresRedeployCommand() *cli.Command {
	return &cli.Command{
		Name:      "redeploy",
		Usage:     "Re-apply CNPG resources for a managed Postgres cluster",
		ArgsUsage: "<storage-id>",
		Action:    handlePostgresRedeploy,
	}
}

func postgresDestroyCommand() *cli.Command {
	return &cli.Command{
		Name:      "destroy",
		Aliases:   []string{"delete", "rm"},
		Usage:     "Destroy a managed Postgres cluster",
		ArgsUsage: "<storage-id>",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "yes", Aliases: []string{"y"}, Usage: "Skip confirmation prompt"},
		},
		Action: handlePostgresDestroy,
	}
}

func postgresUsersCommand() *cli.Command {
	return &cli.Command{
		Name:    "users",
		Aliases: []string{"user"},
		Usage:   "Manage database users",
		Subcommands: []*cli.Command{
			{
				Name:      "list",
				Aliases:   []string{"ls"},
				Usage:     "List database users",
				ArgsUsage: "<storage-id>",
				Action:    handlePostgresUsersList,
			},
			{
				Name:      "create",
				Usage:     "Create a database user",
				ArgsUsage: "<storage-id> <username>",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "superuser", Usage: "Grant SUPERUSER"},
					&cli.BoolFlag{Name: "createdb", Usage: "Grant CREATEDB"},
					&cli.BoolFlag{Name: "createrole", Usage: "Grant CREATEROLE"},
					&cli.BoolFlag{Name: "replication", Usage: "Grant REPLICATION"},
					&cli.BoolFlag{Name: "bypassrls", Usage: "Grant BYPASSRLS"},
					&cli.StringFlag{Name: "comment", Usage: "Role comment"},
				},
				Action: handlePostgresUsersCreate,
			},
			{
				Name:      "delete",
				Aliases:   []string{"rm"},
				Usage:     "Delete a database user",
				ArgsUsage: "<storage-id> <username>",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "yes", Aliases: []string{"y"}, Usage: "Skip confirmation prompt"},
				},
				Action: handlePostgresUsersDelete,
			},
		},
	}
}

func postgresFirewallCommand() *cli.Command {
	return &cli.Command{
		Name:    "firewall",
		Aliases: []string{"fw"},
		Usage:   "Manage Postgres firewall rules",
		Subcommands: []*cli.Command{
			{
				Name:      "list",
				Aliases:   []string{"ls"},
				Usage:     "List firewall rules",
				ArgsUsage: "<storage-id>",
				Action:    handlePostgresFirewallList,
			},
			{
				Name:      "add",
				Usage:     "Add a firewall rule",
				ArgsUsage: "<storage-id>",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "cidr", Usage: "Allowed CIDR, e.g. 203.0.113.10/32", Required: true},
					&cli.StringFlag{Name: "description", Usage: "Rule description", Value: "CLI rule"},
				},
				Action: handlePostgresFirewallAdd,
			},
			{
				Name:      "enable",
				Usage:     "Enable a firewall rule",
				ArgsUsage: "<storage-id> <rule-id>",
				Action:    handlePostgresFirewallEnable,
			},
			{
				Name:      "disable",
				Usage:     "Disable a firewall rule",
				ArgsUsage: "<storage-id> <rule-id>",
				Action:    handlePostgresFirewallDisable,
			},
			{
				Name:      "remove",
				Aliases:   []string{"delete", "rm"},
				Usage:     "Remove a firewall rule",
				ArgsUsage: "<storage-id> <rule-id>",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "yes", Aliases: []string{"y"}, Usage: "Skip confirmation prompt"},
				},
				Action: handlePostgresFirewallRemove,
			},
		},
	}
}

func postgresStorageClassesCommand() *cli.Command {
	return &cli.Command{
		Name:   "storage-classes",
		Usage:  "List available Kubernetes storage classes",
		Action: handlePostgresStorageClasses,
	}
}

func handlePostgresCreate(c *cli.Context) error {
	if c.NArg() < 1 {
		return utils.NewError("cluster name is required. Usage: 1ctl postgres create <name>", nil)
	}

	name := normalizePostgresName(c.Args().First())
	if err := validatePostgresName(name); err != nil {
		return err
	}

	database := c.String("database")
	if database == "" {
		database = strings.ReplaceAll(name, "-", "_")
	}

	opts := api.PostgresCreateOptions{
		Name:           name,
		Database:       database,
		Username:       c.String("user"),
		Version:        c.String("version"),
		Instances:      c.Int("instances"),
		StorageSize:    c.String("storage-size"),
		StorageClass:   c.String("storage-class"),
		WALStorageSize: c.String("wal-storage-size"),
		CPURequest:     c.String("cpu-request"),
		CPULimit:       c.String("cpu"),
		MemoryRequest:  c.String("memory-request"),
		MemoryLimit:    c.String("memory"),
	}
	if opts.Instances < 1 {
		return utils.NewError("instances must be at least 1", nil)
	}

	cluster, err := api.CreatePostgresCluster(opts)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to create Postgres cluster: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(cluster) {
		return nil
	}

	utils.PrintSuccess("Postgres cluster creation started")
	printPostgresCluster(cluster)
	utils.PrintInfo("Use '1ctl postgres status %s' to watch readiness", cluster.StorageID.String())
	return nil
}

func handlePostgresList(c *cli.Context) error {
	clusters, err := api.ListPostgresClusters("")
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list Postgres clusters: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(clusters) {
		return nil
	}
	if len(clusters) == 0 {
		utils.PrintInfo("No Postgres clusters found")
		return nil
	}

	rows := make([][]string, 0, len(clusters))
	for _, cluster := range clusters {
		rows = append(rows, []string{
			cluster.StorageID.String(),
			stringValue(cluster.ClusterName),
			stringValue(cluster.DatabaseName),
			cluster.Version,
			fmt.Sprintf("%d", effectiveInstances(cluster)),
			cluster.StorageSize,
			cluster.StorageClass,
		})
	}
	utils.PrintTable([]string{"ID", "NAME", "DATABASE", "VERSION", "INSTANCES", "SIZE", "CLASS"}, rows)
	return nil
}

func handlePostgresGet(c *cli.Context) error {
	storageID, err := requiredPostgresStorageID(c)
	if err != nil {
		return err
	}
	cluster, err := api.GetPostgresCluster(storageID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get Postgres cluster: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(cluster) {
		return nil
	}
	printPostgresCluster(cluster)
	return nil
}

func handlePostgresStatus(c *cli.Context) error {
	storageID, err := requiredPostgresStorageID(c)
	if err != nil {
		return err
	}
	status, err := api.GetPostgresStatus(storageID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get Postgres status: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(status) {
		return nil
	}
	printPostgresStatus(status)
	return nil
}

func handlePostgresCredentials(c *cli.Context) error {
	storageID, err := requiredPostgresStorageID(c)
	if err != nil {
		return err
	}
	creds, err := api.GetPostgresCredentials(storageID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get Postgres credentials: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(creds) {
		return nil
	}
	printPostgresCredentials(creds)
	return nil
}

func handlePostgresConnect(c *cli.Context) error {
	storageID, err := requiredPostgresStorageID(c)
	if err != nil {
		return err
	}
	creds, err := api.GetPostgresCredentials(storageID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get Postgres credentials: %s", err.Error()), nil)
	}
	uri := creds.URI
	if uri == "" {
		uri = creds.InternalURI
	}
	if uri == "" {
		return utils.NewError("no connection URI available for this Postgres cluster", nil)
	}

	cmd := exec.Command("psql", uri)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return utils.NewError(fmt.Sprintf("failed to run psql: %s", err.Error()), nil)
	}
	return nil
}

func handlePostgresProxy(c *cli.Context) error {
	storageID, err := requiredPostgresStorageID(c)
	if err != nil {
		return err
	}
	creds, err := api.GetPostgresCredentials(storageID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get Postgres credentials: %s", err.Error()), nil)
	}

	remoteHost := creds.Host
	remotePort := creds.Port
	if remoteHost == "" || remotePort == "" {
		return utils.NewError("no remote host/port available for this Postgres cluster", nil)
	}

	localAddr := net.JoinHostPort(c.String("bind-addr"), c.String("local-port"))
	remoteAddr := net.JoinHostPort(remoteHost, remotePort)
	return runTCPProxy(localAddr, remoteAddr)
}

func handlePostgresRedeploy(c *cli.Context) error {
	storageID, err := requiredPostgresStorageID(c)
	if err != nil {
		return err
	}
	if err := api.RedeployPostgresCluster(storageID); err != nil {
		return utils.NewError(fmt.Sprintf("failed to redeploy Postgres cluster: %s", err.Error()), nil)
	}
	utils.PrintSuccess("Postgres redeploy started")
	return nil
}

func handlePostgresDestroy(c *cli.Context) error {
	storageID, err := requiredPostgresStorageID(c)
	if err != nil {
		return err
	}
	if !utils.Confirm(fmt.Sprintf("Destroy Postgres cluster %s and delete its data?", storageID), c.Bool("yes")) {
		fmt.Println("Aborted.")
		return nil
	}
	if err := api.DeletePostgresCluster(storageID); err != nil {
		return utils.NewError(fmt.Sprintf("failed to destroy Postgres cluster: %s", err.Error()), nil)
	}
	utils.PrintSuccess("Postgres cluster destroy started")
	return nil
}

func handlePostgresUsersList(c *cli.Context) error {
	storageID, err := requiredPostgresStorageID(c)
	if err != nil {
		return err
	}
	users, err := api.ListPostgresUsers(storageID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list Postgres users: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(users) {
		return nil
	}
	if len(users) == 0 {
		utils.PrintInfo("No database users found")
		return nil
	}

	rows := make([][]string, 0, len(users))
	for _, user := range users {
		rows = append(rows, []string{
			user.UserID.String(),
			user.Username,
			strings.Join(user.RoleAttributes, ","),
			user.SecretName,
		})
	}
	utils.PrintTable([]string{"ID", "USERNAME", "ATTRIBUTES", "SECRET"}, rows)
	return nil
}

func handlePostgresUsersCreate(c *cli.Context) error {
	if c.NArg() < 2 {
		return utils.NewError("usage: 1ctl postgres users create <storage-id> <username>", nil)
	}
	storageID := c.Args().Get(0)
	username := c.Args().Get(1)
	if username == "" {
		return utils.NewError("username is required", nil)
	}

	resp, err := api.CreatePostgresUser(storageID, api.CreateDatabaseUserRequest{
		Username:       username,
		RoleAttributes: postgresRoleAttributes(c),
		Comment:        c.String("comment"),
	})
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to create Postgres user: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(resp) {
		return nil
	}
	utils.PrintSuccess("Database user created")
	utils.PrintStatusLine("Username", resp.User.Username)
	utils.PrintStatusLine("Password", resp.Password)
	utils.PrintStatusLine("Secret", resp.User.SecretName)
	if resp.ReconciliationStatus != "" {
		utils.PrintStatusLine("Role status", resp.ReconciliationStatus)
	}
	if resp.ReadinessMessage != "" {
		utils.PrintInfo("%s", resp.ReadinessMessage)
	}
	return nil
}

func handlePostgresUsersDelete(c *cli.Context) error {
	if c.NArg() < 2 {
		return utils.NewError("usage: 1ctl postgres users delete <storage-id> <username>", nil)
	}
	storageID := c.Args().Get(0)
	username := c.Args().Get(1)
	if !utils.Confirm(fmt.Sprintf("Delete database user %s from Postgres cluster %s?", username, storageID), c.Bool("yes")) {
		fmt.Println("Aborted.")
		return nil
	}
	if err := api.DeletePostgresUser(storageID, username); err != nil {
		return utils.NewError(fmt.Sprintf("failed to delete Postgres user: %s", err.Error()), nil)
	}
	utils.PrintSuccess("Database user deleted")
	return nil
}

func handlePostgresFirewallList(c *cli.Context) error {
	storageID, err := requiredPostgresStorageID(c)
	if err != nil {
		return err
	}
	rules, err := api.ListPostgresFirewallRules(storageID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list firewall rules: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(rules) {
		return nil
	}
	if len(rules) == 0 {
		utils.PrintInfo("No firewall rules found")
		return nil
	}

	rows := make([][]string, 0, len(rules))
	for _, rule := range rules {
		rows = append(rows, []string{
			rule.RuleID.String(),
			rule.Cidr,
			boolStatus(rule.Enabled),
			rule.Description,
		})
	}
	utils.PrintTable([]string{"ID", "CIDR", "ENABLED", "DESCRIPTION"}, rows)
	return nil
}

func handlePostgresFirewallAdd(c *cli.Context) error {
	storageID, err := requiredPostgresStorageID(c)
	if err != nil {
		return err
	}
	rule, err := api.CreatePostgresFirewallRule(storageID, api.CreateFirewallRuleRequest{
		Cidr:        c.String("cidr"),
		Description: c.String("description"),
	})
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to add firewall rule: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(rule) {
		return nil
	}
	utils.PrintSuccess("Firewall rule added")
	utils.PrintStatusLine("Rule ID", rule.RuleID.String())
	utils.PrintStatusLine("CIDR", rule.Cidr)
	return nil
}

func handlePostgresFirewallEnable(c *cli.Context) error {
	return handlePostgresFirewallSetEnabled(c, true)
}

func handlePostgresFirewallDisable(c *cli.Context) error {
	return handlePostgresFirewallSetEnabled(c, false)
}

func handlePostgresFirewallSetEnabled(c *cli.Context, enabled bool) error {
	if c.NArg() < 2 {
		return utils.NewError("usage: 1ctl postgres firewall enable|disable <storage-id> <rule-id>", nil)
	}
	storageID := c.Args().Get(0)
	ruleID := c.Args().Get(1)
	rule, err := api.UpdatePostgresFirewallRule(storageID, ruleID, api.UpdateFirewallRuleRequest{Enabled: &enabled})
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to update firewall rule: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(rule) {
		return nil
	}
	utils.PrintSuccess("Firewall rule updated")
	utils.PrintStatusLine("Rule ID", rule.RuleID.String())
	utils.PrintStatusLine("Enabled", boolStatus(rule.Enabled))
	return nil
}

func handlePostgresFirewallRemove(c *cli.Context) error {
	if c.NArg() < 2 {
		return utils.NewError("usage: 1ctl postgres firewall remove <storage-id> <rule-id>", nil)
	}
	storageID := c.Args().Get(0)
	ruleID := c.Args().Get(1)
	if !utils.Confirm(fmt.Sprintf("Remove firewall rule %s from Postgres cluster %s?", ruleID, storageID), c.Bool("yes")) {
		fmt.Println("Aborted.")
		return nil
	}
	if err := api.DeletePostgresFirewallRule(storageID, ruleID); err != nil {
		return utils.NewError(fmt.Sprintf("failed to remove firewall rule: %s", err.Error()), nil)
	}
	utils.PrintSuccess("Firewall rule removed")
	return nil
}

func handlePostgresStorageClasses(c *cli.Context) error {
	classes, err := api.ListStorageClasses()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list storage classes: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(classes) {
		return nil
	}
	if len(classes) == 0 {
		utils.PrintInfo("No storage classes found")
		return nil
	}

	rows := make([][]string, 0, len(classes))
	for _, class := range classes {
		rows = append(rows, []string{class.Name, class.Provisioner, boolStatus(class.IsDefault)})
	}
	utils.PrintTable([]string{"NAME", "PROVISIONER", "DEFAULT"}, rows)
	return nil
}

func requiredPostgresStorageID(c *cli.Context) (string, error) {
	if c.NArg() < 1 {
		return "", utils.NewError("storage ID is required", nil)
	}
	return c.Args().First(), nil
}

func validatePostgresName(name string) error {
	if name == "" {
		return utils.NewError("cluster name cannot be empty", nil)
	}
	if len(name) > 40 {
		return utils.NewError("cluster name must be 40 characters or fewer", nil)
	}
	if !postgresNamePattern.MatchString(name) {
		return utils.NewError("cluster name must contain only lowercase letters, numbers, and hyphens, and must start and end with a letter or number", nil)
	}
	return nil
}

func normalizePostgresName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func printPostgresCluster(cluster *api.StorageConfig) {
	if cluster == nil {
		return
	}
	utils.PrintHeader("Postgres Cluster")
	utils.PrintStatusLine("Storage ID", cluster.StorageID.String())
	utils.PrintStatusLine("Name", stringValue(cluster.ClusterName))
	utils.PrintStatusLine("Database", stringValue(cluster.DatabaseName))
	utils.PrintStatusLine("Username", stringValue(cluster.Username))
	utils.PrintStatusLine("Namespace", cluster.Namespace)
	utils.PrintStatusLine("Version", cluster.Version)
	utils.PrintStatusLine("Instances", fmt.Sprintf("%d", effectiveInstances(*cluster)))
	utils.PrintStatusLine("Storage", fmt.Sprintf("%s (%s)", cluster.StorageSize, cluster.StorageClass))
	utils.PrintStatusLine("Resources", fmt.Sprintf("%s-%s CPU, %s-%s memory", cluster.CPURequest, cluster.CPULimit, cluster.MemoryRequest, cluster.MemoryLimit))
}

func printPostgresStatus(status *api.PostgresStatus) {
	if status == nil {
		return
	}
	utils.PrintHeader("Postgres Status")
	utils.PrintStatusLine("Status", status.Status)
	utils.PrintStatusLine("Cluster phase", status.ClusterPhase)
	utils.PrintStatusLine("Cluster exists", boolStatus(status.ClusterExists))
	utils.PrintStatusLine("Instances", fmt.Sprintf("%d/%d ready", status.ReadyInstances, status.TotalInstances))
	if status.Primary != "" {
		utils.PrintStatusLine("Primary", status.Primary)
	}
	utils.PrintStatusLine("Pooler", fmt.Sprintf("%d/%d ready", status.PoolerReadyReplicas, status.PoolerDesiredReplicas))
	utils.PrintStatusLine("External access", boolStatus(status.ExternalAccessible))
}

func printPostgresCredentials(creds *api.PostgresCredentials) {
	if creds == nil {
		return
	}
	utils.PrintHeader("Postgres Credentials")
	utils.PrintStatusLine("Username", creds.Username)
	utils.PrintStatusLine("Password", creds.Password)
	utils.PrintStatusLine("Database", creds.DBName)
	utils.PrintStatusLine("Host", creds.Host)
	utils.PrintStatusLine("Port", creds.Port)
	utils.PrintStatusLine("URI", creds.URI)
	if creds.InternalURI != "" {
		utils.PrintStatusLine("Internal URI", creds.InternalURI)
	}
	if creds.ExternalURI != "" {
		utils.PrintStatusLine("External URI", creds.ExternalURI)
	}
}

func effectiveInstances(cluster api.StorageConfig) int {
	if cluster.Instances != nil && *cluster.Instances > 0 {
		return *cluster.Instances
	}
	return cluster.Replicas
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func boolStatus(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func postgresRoleAttributes(c *cli.Context) []string {
	attrs := []string{}
	for _, attr := range []struct {
		flag string
		name string
	}{
		{"superuser", "SUPERUSER"},
		{"createdb", "CREATEDB"},
		{"createrole", "CREATEROLE"},
		{"replication", "REPLICATION"},
		{"bypassrls", "BYPASSRLS"},
	} {
		if c.Bool(attr.flag) {
			attrs = append(attrs, attr.name)
		}
	}
	return attrs
}

func runTCPProxy(localAddr, remoteAddr string) error {
	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to listen on %s: %s", localAddr, err.Error()), nil)
	}
	defer func() { _ = listener.Close() }()

	utils.PrintSuccess("Proxy listening on %s -> %s", localAddr, remoteAddr)
	utils.PrintInfo("Connect with: psql postgresql://USER:PASSWORD@%s/DBNAME?sslmode=require", localAddr)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)

	errCh := make(chan error, 1)
	go func() {
		for {
			conn, acceptErr := listener.Accept()
			if acceptErr != nil {
				errCh <- acceptErr
				return
			}
			go proxyConnection(conn, remoteAddr)
		}
	}()

	select {
	case <-signals:
		fmt.Println()
		return nil
	case err := <-errCh:
		return utils.NewError(fmt.Sprintf("proxy stopped: %s", err.Error()), nil)
	}
}

func proxyConnection(local net.Conn, remoteAddr string) {
	defer func() { _ = local.Close() }()

	remote, err := net.Dial("tcp", remoteAddr)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to connect to %s: %s\n", remoteAddr, err.Error())
		return
	}
	defer func() { _ = remote.Close() }()

	done := make(chan struct{}, 2)
	go func() {
		_, _ = io.Copy(remote, local)
		done <- struct{}{}
	}()
	go func() {
		_, _ = io.Copy(local, remote)
		done <- struct{}{}
	}()
	<-done
}
