package postgres

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"1ctl/internal/api"
	satuskyctx "1ctl/internal/context"
	"1ctl/internal/utils"
)

func handlePostgresCreate(ctx context.Context, in postgresCreateInput) error {
	name := normalizePostgresName(in.Name)
	if err := validatePostgresName(name); err != nil {
		return err
	}

	database := in.Database
	if database == "" {
		database = strings.ReplaceAll(name, "-", "_")
	}

	opts := api.PostgresCreateOptions{
		Name:           name,
		Database:       database,
		Username:       in.User,
		Version:        in.Version,
		Instances:      in.Instances,
		StorageSize:    in.StorageSize,
		StorageClass:   in.StorageClass,
		WALStorageSize: in.WALSize,
		CPURequest:     in.CPURequest,
		CPULimit:       in.CPULimit,
		MemoryRequest:  in.MemRequest,
		MemoryLimit:    in.MemLimit,
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

func handlePostgresList(ctx context.Context) error {
	clusters, err := api.ListPostgresClusters("")
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list Postgres clusters: %s", err.Error()), nil)
	}
	if utils.PrintListOrJSON(clusters, "No Postgres clusters found") {
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

func handlePostgresGet(ctx context.Context, in postgresStorageIDInput) error {
	storageID, err := resolvePostgresStorageID(in.StorageID)
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

func handlePostgresStatus(ctx context.Context, in postgresStorageIDInput) error {
	storageID, err := resolvePostgresStorageID(in.StorageID)
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

func handlePostgresCredentials(ctx context.Context, in postgresStorageIDInput) error {
	storageID, err := resolvePostgresStorageID(in.StorageID)
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

func handlePostgresConnect(ctx context.Context, in postgresConnectInput) error {
	storageID, err := resolvePostgresStorageID(in.StorageID)
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

	psqlCmd := exec.Command("psql", uri)
	psqlCmd.Stdin = os.Stdin
	psqlCmd.Stdout = os.Stdout
	psqlCmd.Stderr = os.Stderr
	if err := psqlCmd.Run(); err != nil {
		return utils.NewError(fmt.Sprintf("failed to run psql: %s", err.Error()), nil)
	}
	return nil
}

func handlePostgresProxy(ctx context.Context, in postgresProxyInput) error {
	storageID, err := resolvePostgresStorageID(in.StorageID)
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

	localAddr := net.JoinHostPort(in.BindAddr, in.LocalPort)
	remoteAddr := net.JoinHostPort(remoteHost, remotePort)
	return runTCPProxy(localAddr, remoteAddr)
}

func handlePostgresRedeploy(ctx context.Context, in postgresStorageIDInput) error {
	storageID, err := resolvePostgresStorageID(in.StorageID)
	if err != nil {
		return err
	}
	if err := api.RedeployPostgresCluster(storageID); err != nil {
		return utils.NewError(fmt.Sprintf("failed to redeploy Postgres cluster: %s", err.Error()), nil)
	}
	utils.PrintSuccess("Postgres redeploy started")
	return nil
}

func handlePostgresDestroy(ctx context.Context, in postgresDestroyInput) error {
	storageID, err := resolvePostgresStorageID(in.StorageID)
	if err != nil {
		return err
	}
	if !utils.Confirm(fmt.Sprintf("Destroy Postgres cluster %s and delete its data?", storageID), in.Yes) {
		fmt.Println("Aborted.")
		return nil
	}
	if err := api.DeletePostgresCluster(storageID); err != nil {
		return utils.NewError(fmt.Sprintf("failed to destroy Postgres cluster: %s", err.Error()), nil)
	}
	utils.PrintSuccess("Postgres cluster destroy started")
	return nil
}

func handlePostgresUsersList(ctx context.Context, in postgresStorageIDInput) error {
	storageID, err := resolvePostgresStorageID(in.StorageID)
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

func handlePostgresUsersCreate(ctx context.Context, in postgresUsersCreateInput) error {
	storageID, err := resolvePostgresStorageID(in.StorageID)
	if err != nil {
		return err
	}

	resp, err := api.CreatePostgresUser(storageID, api.CreateDatabaseUserRequest{
		Username:       in.Username,
		RoleAttributes: postgresRoleAttributes(in),
		Comment:        in.Comment,
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

func handlePostgresUsersDelete(ctx context.Context, in postgresUsersDeleteInput) error {
	storageID, err := resolvePostgresStorageID(in.StorageID)
	if err != nil {
		return err
	}
	if !utils.Confirm(fmt.Sprintf("Delete database user %s from Postgres cluster %s?", in.Username, storageID), in.Yes) {
		fmt.Println("Aborted.")
		return nil
	}
	if err := api.DeletePostgresUser(storageID, in.Username); err != nil {
		return utils.NewError(fmt.Sprintf("failed to delete Postgres user: %s", err.Error()), nil)
	}
	utils.PrintSuccess("Database user deleted")
	return nil
}

func handlePostgresFirewallList(ctx context.Context, in postgresStorageIDInput) error {
	storageID, err := resolvePostgresStorageID(in.StorageID)
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

func handlePostgresFirewallAdd(ctx context.Context, in postgresFirewallAddInput) error {
	storageID, err := resolvePostgresStorageID(in.StorageID)
	if err != nil {
		return err
	}
	rule, err := api.CreatePostgresFirewallRule(storageID, api.CreateFirewallRuleRequest{
		Cidr:        in.CIDR,
		Description: in.Description,
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

func handlePostgresFirewallSetEnabled(ctx context.Context, storageID, ruleID string, enabled bool) error {
	sid, err := resolvePostgresStorageID(storageID)
	if err != nil {
		return err
	}
	rule, err := api.UpdatePostgresFirewallRule(sid, ruleID, api.UpdateFirewallRuleRequest{Enabled: &enabled})
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

func handlePostgresFirewallRemove(ctx context.Context, in postgresFirewallRemoveInput) error {
	storageID, err := resolvePostgresStorageID(in.StorageID)
	if err != nil {
		return err
	}
	if !utils.Confirm(fmt.Sprintf("Remove firewall rule %s from Postgres cluster %s?", in.RuleID, storageID), in.Yes) {
		fmt.Println("Aborted.")
		return nil
	}
	if err := api.DeletePostgresFirewallRule(storageID, in.RuleID); err != nil {
		return utils.NewError(fmt.Sprintf("failed to remove firewall rule: %s", err.Error()), nil)
	}
	utils.PrintSuccess("Firewall rule removed")
	return nil
}

func handlePostgresStorageClasses(ctx context.Context) error {
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

// --- Shared helpers -----------------------------------------------------

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

func resolvePostgresStorageID(arg string) (string, error) {
	if arg == "" {
		return "", utils.NewError("storage ID is required", nil)
	}
	// If it already looks like a UUID, return as-is.
	if _, uuidErr := api.ParseUUID(arg); uuidErr == nil {
		return arg, nil
	}
	// Resolve by name.
	ns := satuskyctx.GetCurrentNamespace()
	if ns == "" {
		return "", utils.NewError("not authenticated — run '1ctl auth login' first", nil)
	}
	clusters, err := api.ListPostgresClusters(ns)
	if err != nil {
		return "", utils.NewError(fmt.Sprintf("failed to list postgres clusters: %s", err.Error()), nil)
	}
	for _, cluster := range clusters {
		if cluster.ClusterName != nil && strings.EqualFold(*cluster.ClusterName, arg) {
			return cluster.StorageID.String(), nil
		}
	}
	return "", utils.NewError(fmt.Sprintf("postgres cluster %q not found — pass the storage ID from '1ctl postgres list'", arg), nil)
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

func postgresRoleAttributes(in postgresUsersCreateInput) []string {
	attrs := []string{}
	if in.Superuser {
		attrs = append(attrs, "SUPERUSER")
	}
	if in.Createdb {
		attrs = append(attrs, "CREATEDB")
	}
	if in.Createrole {
		attrs = append(attrs, "CREATEROLE")
	}
	if in.Replication {
		attrs = append(attrs, "REPLICATION")
	}
	if in.BypassRLS {
		attrs = append(attrs, "BYPASSRLS")
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
