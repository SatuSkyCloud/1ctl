package commands

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"1ctl/internal/api"
	satuskyctx "1ctl/internal/context"
	"1ctl/internal/utils"
	"1ctl/internal/validator"

	"github.com/google/uuid"
	"github.com/urfave/cli/v3"
)

type domainListEntry struct {
	DomainName string            `json:"domain_name"`
	AppLabel   string            `json:"app_label"`
	Namespace  string            `json:"namespace"`
	DnsConfig  api.DnsConfigType `json:"dns_config"`
	Kind       string            `json:"kind"`
	CreatedAt  string            `json:"created_at"`
}

// DomainsCommand exposes app-oriented domain/TLS workflows over the same
// backend routes that `ingress` uses, but resolves deployment, service,
// namespace, and ingress IDs internally from the user's app name. Users
// don't see Kubernetes Ingress or cert-manager Issuer concepts.
func DomainsCommand() *cli.Command {
	return &cli.Command{
		Name:    "domains",
		Aliases: []string{"domain"},
		Usage:   "Add, list, and inspect custom domains for your apps",
		Commands: []*cli.Command{
			domainsListCommand(),
			domainsAddCommand(),
			domainsRemoveCommand(),
			domainsCheckCommand(),
			domainsSetupCommand(),
			domainsAvailableCommand(),
			domainsSearchCommand(),
			domainsManagedCommand(),
			domainsDNSCommand(),
			domainsPurchaseCommand(),
			domainsPurchaseStatusCommand(),
		},
	}
}

func domainsListCommand() *cli.Command {
	return &cli.Command{
		Name:   "list",
		Usage:  "List all domains in the current organization",
		Action: handleDomainsList,
	}
}

func domainsAddCommand() *cli.Command {
	return &cli.Command{
		Name:      "add",
		Usage:     "Add a custom domain to an app",
		ArgsUsage: "<domain>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "app",
				Usage:    "App name (the value of [app] name in satusky.toml, or --name on deploy)",
				Required: true,
			},
			&cli.IntFlag{
				Name:  "port",
				Usage: "Target port on the deployment",
				Value: 8080,
			},
			&cli.BoolFlag{
				Name:  "custom-dns",
				Usage: "Treat the hostname as an external custom domain",
			},
			&cli.BoolFlag{
				Name:  "wait",
				Usage: "Wait for DNS and TLS to become ready before returning",
				Value: true,
			},
			&cli.BoolFlag{
				Name:  "no-wait",
				Usage: "Skip waiting — return immediately after attaching",
			},
			&cli.BoolFlag{
				Name:  "with-www",
				Usage: "Also configure a www redirect for apex domains when supported",
			},
		},
		Action: handleDomainsAdd,
	}
}

func domainsRemoveCommand() *cli.Command {
	return &cli.Command{
		Name:      "remove",
		Aliases:   []string{"rm"},
		Usage:     "Remove a custom domain from an app",
		ArgsUsage: "<domain>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "app",
				Usage:    "App name (the value of [app] name in satusky.toml)",
				Required: true,
			},
			&cli.BoolFlag{
				Name:    "yes",
				Aliases: []string{"y"},
				Usage:   "Skip confirmation prompt",
			},
		},
		Action: handleDomainsRemove,
	}
}

func domainsCheckCommand() *cli.Command {
	return &cli.Command{
		Name:      "check",
		Usage:     "Check backend, route, DNS, TLS, and HTTP status for a domain",
		ArgsUsage: "<domain>",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "probe",
				Usage: "Run an HTTP reachability probe",
			},
		},
		Action: handleDomainsCheck,
	}
}

func domainsSetupCommand() *cli.Command {
	return &cli.Command{
		Name:      "setup",
		Usage:     "Show exact DNS setup instructions for a domain",
		ArgsUsage: "<domain>",
		Action:    handleDomainsSetup,
	}
}

func handleDomainsList(ctx context.Context, cmd *cli.Command) error {
	if _, err := satuskyctx.GetCurrentNamespaceOrError(); err != nil {
		return err
	}
	ingresses, err := api.ListIngresses()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list domains: %s", err.Error()), nil)
	}

	entries := make([]domainListEntry, 0, len(ingresses))
	for _, ing := range ingresses {
		entries = append(entries, domainListEntry{
			DomainName: ing.DomainName,
			AppLabel:   ing.AppLabel,
			Namespace:  ing.Namespace,
			DnsConfig:  ing.DnsConfig,
			Kind:       "primary",
			CreatedAt:  api.FormatTimeAgo(ing.CreatedAt),
		})
		aliases, aliasErr := api.ListDomainAliases(ing.IngressID.String())
		if aliasErr != nil {
			continue
		}
		for _, alias := range aliases {
			if alias.IsRedirect {
				continue
			}
			entries = append(entries, domainListEntry{
				DomainName: alias.DomainName,
				AppLabel:   ing.AppLabel,
				Namespace:  ing.Namespace,
				DnsConfig:  alias.DnsConfig,
				Kind:       "custom",
				CreatedAt:  api.FormatTimeAgo(alias.CreatedAt),
			})
		}
	}

	if len(entries) == 0 {
		utils.PrintInfo("No domains yet. Add one with: 1ctl domains add <domain> --app <app>")
		return nil
	}

	if utils.TryPrintJSON(entries) {
		return nil
	}

	headers := []string{"DOMAIN", "APP", "KIND", "TLS", "CREATED"}
	rows := make([][]string, 0, len(entries))
	for _, entry := range entries {
		tls := "platform"
		if entry.DnsConfig == api.DnsConfigCustom {
			tls = "letsencrypt"
		}
		rows = append(rows, []string{
			entry.DomainName,
			entry.AppLabel,
			entry.Kind,
			tls,
			entry.CreatedAt,
		})
	}
	utils.PrintTable(headers, rows)
	return nil
}

func handleDomainsAdd(ctx context.Context, cmd *cli.Command) error {
	if cmd.NArg() < 1 {
		return utils.NewError("domain is required. Usage: 1ctl domains add <domain> --app <app>", nil)
	}
	domain, err := normalizeDomainArg(cmd.Args().First())
	if err != nil {
		return err
	}
	appName := cmd.String("app")

	namespace, err := satuskyctx.GetCurrentNamespaceOrError()
	if err != nil {
		return err
	}

	// Resolve the deployment by app label so the user doesn't supply UUIDs.
	dep, err := api.GetDeploymentByAppLabel(namespace, appName)
	if err != nil {
		return utils.NewError(fmt.Sprintf("could not find an app named %q in the current organization: %s", appName, err.Error()), nil)
	}

	// Find the matching Service for this deployment so we can wire the ingress.
	services, err := api.ListServices()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list services: %s", err.Error()), nil)
	}
	var serviceID string
	for _, s := range services {
		if s.DeploymentID == dep.DeploymentID {
			serviceID = s.ServiceID.String()
			break
		}
	}
	if serviceID == "" {
		return utils.NewError(fmt.Sprintf("no service found for app %q — has the app been deployed?", appName), nil)
	}

	port, err := api.SafeInt32(cmd.Int("port"))
	if err != nil {
		return utils.NewError("invalid --port value", err)
	}

	existingIngress, _ := api.GetIngressByDeploymentID(dep.DeploymentID.String())

	// Auto-pick TLS strategy: external hostnames are independent custom-domain
	// aliases. Platform-owned hostnames keep the legacy primary route behavior.
	dnsCfg := api.DnsConfigDefault
	isSatuskyHost := strings.HasSuffix(strings.ToLower(domain), ".satusky.com") || strings.ToLower(domain) == "satusky.com"
	if cmd.Bool("custom-dns") || !isSatuskyHost {
		dnsCfg = api.DnsConfigCustom
	}

	if dnsCfg == api.DnsConfigCustom {
		orgID, err := currentOrgUUID()
		if err != nil {
			return err
		}
		ing := existingIngress
		if ing == nil || ing.IngressID == uuid.Nil {
			defaultDomain, genErr := api.GenerateDomainName(appName)
			if genErr != nil {
				return utils.NewError(fmt.Sprintf("failed to generate default app domain before attaching custom domain: %s", genErr.Error()), nil)
			}
			created, createErr := api.UpsertIngress(api.Ingress{
				DeploymentID: dep.DeploymentID,
				ServiceID:    api.ToUUID(serviceID),
				AppLabel:     appName,
				Namespace:    namespace,
				DomainName:   defaultDomain,
				DnsConfig:    api.DnsConfigDefault,
				Port:         port,
			})
			if createErr != nil {
				return utils.NewError(fmt.Sprintf("failed to create default app route before attaching custom domain: %s", createErr.Error()), nil)
			}
			ing = created
		}
		alias, err := api.AttachDomain(ing.IngressID.String(), api.AttachDomainRequest{
			OrgID:           orgID,
			DomainName:      domain,
			WithWWWRedirect: cmd.Bool("with-www"),
		})
		if err != nil {
			return utils.NewError(fmt.Sprintf("failed to add domain: %s", err.Error()), nil)
		}
		utils.PrintSuccess("Domain %s attached to app %s", alias.DomainName, appName)

		if cmd.Bool("wait") && !cmd.Bool("no-wait") && !cmd.IsSet("no-wait") {
			if err := waitForDomainLive(ing.IngressID.String(), domain, 3*time.Minute); err != nil {
				utils.PrintWarning("Domain is attached but not yet live: %s", err.Error())
				utils.PrintInfo("Check status later: 1ctl domains check %s --probe", domain)
			}
			return nil
		}

		if status, statusErr := api.GetDomainStatus(ing.IngressID.String(), domain, false); statusErr == nil {
			printDomainSetup(status)
		} else {
			utils.PrintInfo("Custom domain: run '1ctl domains setup %s' for exact DNS records.", domain)
		}
		utils.PrintInfo("  1ctl domains check %s", domain)
		return nil
	}

	ingress := api.Ingress{
		DeploymentID: dep.DeploymentID,
		ServiceID:    api.ToUUID(serviceID),
		AppLabel:     appName,
		Namespace:    namespace,
		DomainName:   domain,
		DnsConfig:    dnsCfg,
		Port:         port,
	}
	resp, err := api.UpsertIngress(ingress)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to add domain: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Domain %s attached to app %s", resp.DomainName, appName)
	if dnsCfg == api.DnsConfigCustom {
		if status, statusErr := api.GetDomainStatus(resp.IngressID.String(), domain, false); statusErr == nil {
			printDomainSetup(status)
		} else {
			utils.PrintInfo("Custom domain: run '1ctl domains setup %s' for exact DNS records.", domain)
		}
		utils.PrintInfo("  1ctl domains check %s", domain)
	}
	return nil
}

func handleDomainsRemove(ctx context.Context, cmd *cli.Command) error {
	if cmd.NArg() < 1 {
		return utils.NewError("domain is required. Usage: 1ctl domains remove <domain> --app <app>", nil)
	}
	domain, err := normalizeDomainArg(cmd.Args().First())
	if err != nil {
		return err
	}
	appName := cmd.String("app")

	if _, err := satuskyctx.GetCurrentNamespaceOrError(); err != nil {
		return err
	}

	ing, err := api.GetIngressByDomainName(domain)
	if err != nil {
		return utils.NewError(fmt.Sprintf("no domain %q found in this organization: %s", domain, err.Error()), nil)
	}
	if ing.AppLabel != appName {
		return utils.NewError(fmt.Sprintf("domain %q belongs to app %q, not %q — refusing to remove without explicit match", domain, ing.AppLabel, appName), nil)
	}
	if !utils.Confirm(fmt.Sprintf("Remove domain %s from app %s?", domain, appName), cmd.Bool("yes") || cmd.IsSet("yes") || cmd.IsSet("y")) {
		fmt.Println("Aborted.")
		return nil
	}
	orgID, err := currentOrgUUID()
	if err != nil {
		return err
	}
	if err := api.DetachDomain(ing.IngressID.String(), api.DetachDomainRequest{OrgID: orgID, DomainName: domain}); err != nil {
		return utils.NewError(fmt.Sprintf("failed to remove domain: %s", err.Error()), nil)
	}
	utils.PrintSuccess("Domain %s removed from app %s", domain, appName)
	return nil
}

func handleDomainsCheck(ctx context.Context, cmd *cli.Command) error {
	if cmd.NArg() < 1 {
		return utils.NewError("domain is required. Usage: 1ctl domains check <domain>", nil)
	}
	domain, err := normalizeDomainArg(cmd.Args().First())
	if err != nil {
		return err
	}

	ing, err := api.GetIngressByDomainName(domain)
	if err != nil {
		return printDetachedDomainStatus(domain, err)
	}

	probe := cmd.Bool("probe") || cmd.IsSet("probe")
	status, err := api.GetDomainStatus(ing.IngressID.String(), domain, probe)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to check domain %q: %s", domain, err.Error()), nil)
	}

	if utils.TryPrintJSON(status) {
		return nil
	}

	printDomainStatus(status)
	return nil
}

func handleDomainsSetup(ctx context.Context, cmd *cli.Command) error {
	if cmd.NArg() < 1 {
		return utils.NewError("domain is required. Usage: 1ctl domains setup <domain>", nil)
	}
	domain, err := normalizeDomainArg(cmd.Args().First())
	if err != nil {
		return err
	}

	ing, err := api.GetIngressByDomainName(domain)
	if err != nil {
		return printDetachedDomainSetup(domain, err)
	}
	status, err := api.GetDomainStatus(ing.IngressID.String(), domain, false)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to load setup details for %q: %s", domain, err.Error()), nil)
	}
	if utils.TryPrintJSON(status) {
		return nil
	}

	printDomainSetup(status)
	return nil
}

func printDetachedDomainStatus(domain string, cause error) error {
	if utils.IsJSONOutput() {
		utils.TryPrintJSON(map[string]interface{}{
			"domain_name": domain,
			"attached":    false,
			"message":     cause.Error(),
		})
		return nil
	}
	utils.PrintHeader("Domain %s", domain)
	utils.PrintStatusLine("Backend", "not attached")
	utils.PrintStatusLine("Route", "not attached")
	utils.PrintStatusLine("DNS", "not checked")
	utils.PrintStatusLine("TLS", "unknown")
	utils.PrintStatusLine("HTTP", "not checked")
	utils.PrintInfo("Attach it with: 1ctl domains add %s --app <app>", domain)
	return nil
}

func printDetachedDomainSetup(domain string, cause error) error {
	if utils.IsJSONOutput() {
		utils.TryPrintJSON(map[string]interface{}{
			"domain_name": domain,
			"attached":    false,
			"message":     cause.Error(),
		})
		return nil
	}
	utils.PrintHeader("Domain Setup %s", domain)
	utils.PrintStatusLine("Backend", "not attached")
	utils.PrintInfo("Attach it first with: 1ctl domains add %s --app <app>", domain)
	return nil
}

func printDomainStatus(status *api.DomainStatusResponse) {
	utils.PrintHeader("Domain %s", status.DomainName)
	utils.PrintStatusLine("Backend", fmt.Sprintf("attached to %s in %s", status.AppLabel, status.Namespace))
	utils.PrintStatusLine("Route", domainRouteText(status.Route))
	utils.PrintStatusLine("DNS", domainDNSText(status.DNS))
	utils.PrintStatusLine("TLS", domainTLSText(status.TLS))
	utils.PrintStatusLine("HTTP", domainHTTPText(status.Reachability))
	if status.DNS.Status != api.DNSStatusResolved {
		utils.PrintInfo("Run setup details with: 1ctl domains setup %s", status.DomainName)
	}
}

func printDomainSetup(status *api.DomainStatusResponse) {
	utils.PrintHeader("Domain Setup %s", status.DomainName)
	utils.PrintStatusLine("App", status.AppLabel)
	utils.PrintStatusLine("Namespace", status.Namespace)
	utils.PrintStatusLine("Current DNS", domainDNSText(status.DNS))
	utils.PrintStatusLine("TLS", domainTLSText(status.TLS))

	if status.DNS.ExpectedIP == "" {
		utils.PrintWarning("Backend did not return an expected DNS target yet.")
		return
	}

	recordType := "A"
	if net.ParseIP(status.DNS.ExpectedIP) == nil {
		recordType = "CNAME"
	}
	utils.PrintHeader("Required DNS Records")
	utils.PrintStatusLine("Type", recordType)
	utils.PrintStatusLine("Name", status.DomainName)
	utils.PrintStatusLine("Value", status.DNS.ExpectedIP)
	utils.PrintInfo("Next check: 1ctl domains check %s --probe", status.DomainName)
}

func domainRouteText(status api.DomainRouteStatus) string {
	if !status.Attached {
		if status.Message != "" {
			return "not attached: " + status.Message
		}
		return "not attached"
	}
	if status.ResourceKind == "" && status.ResourceName == "" {
		return "attached"
	}
	return fmt.Sprintf("attached to %s/%s", status.ResourceKind, status.ResourceName)
}

func domainDNSText(status api.DNSStatusResponse) string {
	parts := []string{string(status.Status)}
	if len(status.ResolvedIPs) > 0 {
		parts = append(parts, "resolved "+strings.Join(status.ResolvedIPs, ", "))
	}
	if status.ExpectedIP != "" {
		parts = append(parts, "expected "+status.ExpectedIP)
	}
	if status.Message != "" {
		parts = append(parts, status.Message)
	}
	return strings.Join(parts, " - ")
}

func domainTLSText(status api.TLSStatusResponse) string {
	if status.Status == "" {
		return "unknown"
	}
	if status.ExpiresAt != nil {
		return fmt.Sprintf("%s, expires %s", status.Status, status.ExpiresAt.Format("2006-01-02"))
	}
	if status.Message != "" {
		return fmt.Sprintf("%s - %s", status.Status, status.Message)
	}
	return string(status.Status)
}

func domainHTTPText(status api.DomainReachabilityStatus) string {
	if !status.Checked {
		if status.Message != "" {
			return "not checked - " + status.Message
		}
		return "not checked"
	}
	if status.Reachable {
		return fmt.Sprintf("reachable %s %d", status.URL, status.StatusCode)
	}
	if status.Message != "" {
		return "not reachable - " + status.Message
	}
	return "not reachable"
}

func waitForDomainLive(ingressID, domain string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	utils.PrintInfo("Waiting for domain to become live (DNS + TLS + HTTP)...")

	lastDNS := ""
	lastTLS := ""
	lastHTTP := ""
	i := 0

	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out after %v", timeout)
		}

		status, err := api.GetDomainStatus(ingressID, domain, true)
		if err != nil {
			time.Sleep(time.Second)
			continue
		}

		dnsText := fmt.Sprintf("resolved %s", strings.Join(status.DNS.ResolvedIPs, ", "))
		tlsText := string(status.TLS.Status)
		httpText := "not checked"
		if status.Reachability.Checked {
			if status.Reachability.Reachable {
				httpText = fmt.Sprintf("HTTP %d", status.Reachability.StatusCode)
			} else {
				httpText = "not reachable"
			}
		}

		// Print status updates only when they change
		if dnsText != lastDNS {
			fmt.Printf("\r   %s DNS: %s\n", spinner[i%len(spinner)], dnsText)
			lastDNS = dnsText
		}
		if tlsText != lastTLS {
			fmt.Printf("\r   %s TLS: %s\n", spinner[i%len(spinner)], tlsText)
			lastTLS = tlsText
		}
		if httpText != lastHTTP {
			fmt.Printf("\r   %s HTTP: %s\n", spinner[i%len(spinner)], httpText)
			lastHTTP = httpText
		}

		// Success condition: DNS resolved + TLS active + HTTP reachable
		if status.DNS.Status == api.DNSStatusResolved &&
			status.TLS.Status == api.TLSStatusActive &&
			status.Reachability.Checked &&
			status.Reachability.Reachable {
			fmt.Println()
			utils.PrintSuccess("Domain is live!")
			utils.PrintStatusLine("URL", fmt.Sprintf("https://%s", domain))
			utils.PrintStatusLine("HTTP", fmt.Sprintf("%d %s", status.Reachability.StatusCode, status.Reachability.Message))
			if status.TLS.ExpiresAt != nil {
				utils.PrintStatusLine("TLS expires", status.TLS.ExpiresAt.Format("2006-01-02"))
			}
			return nil
		}

		i++
		<-ticker.C
	}
}

func normalizeDomainArg(domain string) (string, error) {
	domain = strings.ToLower(strings.TrimSpace(domain))
	domain = strings.TrimSuffix(domain, ".")
	if strings.HasPrefix(domain, "http://") || strings.HasPrefix(domain, "https://") || strings.Contains(domain, "/") {
		return "", utils.NewError("domain must be a hostname, not a URL", nil)
	}
	if strings.HasPrefix(domain, "*.") {
		return "", utils.NewError("wildcard domains are not supported yet because SatuSky custom-domain TLS currently uses HTTP-01 validation", nil)
	}
	if err := validator.ValidateDomain(domain); err != nil {
		return "", err
	}
	return domain, nil
}

func currentOrgUUID() (uuid.UUID, error) {
	orgID := strings.TrimSpace(satuskyctx.GetCurrentOrgID())
	if orgID == "" {
		return uuid.Nil, utils.NewError("no current organization ID is set. Run: 1ctl org switch --id <org-id>", nil)
	}
	parsed, err := api.ParseUUID(orgID)
	if err != nil {
		return uuid.Nil, utils.NewError("current organization ID is invalid. Run: 1ctl org switch --id <org-id>", err)
	}
	return parsed, nil
}
