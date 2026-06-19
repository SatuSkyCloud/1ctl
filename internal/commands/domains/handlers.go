package domains

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
)

// --- Domain listing -----------------------------------------------------

func handleDomainsList(ctx context.Context) error {
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
			CreatedAt:  utils.FormatTimeAgo(ing.CreatedAt),
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
				CreatedAt:  utils.FormatTimeAgo(alias.CreatedAt),
			})
		}
	}

	if utils.PrintListOrJSON(entries, "No domains yet. Add one with: 1ctl domains add --domain <domain> --app <app>") {
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

// --- Domain add ---------------------------------------------------------

func handleDomainsAdd(ctx context.Context, in domainsAddInput) error {
	domain, err := normalizeDomainArg(in.Domain)
	if err != nil {
		return err
	}
	appName := in.App

	namespace, err := satuskyctx.GetCurrentNamespaceOrError()
	if err != nil {
		return err
	}

	dep, err := api.GetDeploymentByAppLabel(namespace, appName)
	if err != nil {
		return utils.NewError(fmt.Sprintf("could not find an app named %q in the current organization: %s", appName, err.Error()), nil)
	}

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

	port, err := api.SafeInt32(in.Port)
	if err != nil {
		return utils.NewError("invalid --port value", err)
	}

	existingIngress, _ := api.GetIngressByDeploymentID(dep.DeploymentID.String())

	dnsCfg := api.DnsConfigDefault
	isSatuskyHost := strings.HasSuffix(strings.ToLower(domain), ".satusky.com") || strings.ToLower(domain) == "satusky.com"
	if in.CustomDNS || !isSatuskyHost {
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
			WithWWWRedirect: in.WithWWW,
		})
		if err != nil {
			return utils.NewError(fmt.Sprintf("failed to add domain: %s", err.Error()), nil)
		}
		utils.PrintSuccess("Domain %s attached to app %s", alias.DomainName, appName)

		if in.Wait && !in.NoWait {
			if err := waitForDomainLive(ing.IngressID.String(), domain, 3*time.Minute); err != nil {
				utils.PrintWarning("Domain is attached but not yet live: %s", err.Error())
				utils.PrintInfo("Check status later: 1ctl domains check --domain %s --probe", domain)
			}
			return nil
		}

		if status, statusErr := api.GetDomainStatus(ing.IngressID.String(), domain, false); statusErr == nil {
			printDomainSetup(status)
		} else {
			utils.PrintInfo("Custom domain: run '1ctl domains setup --domain %s' for exact DNS records.", domain)
		}
		utils.PrintInfo("  1ctl domains check --domain %s", domain)
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
			utils.PrintInfo("Custom domain: run '1ctl domains setup --domain %s' for exact DNS records.", domain)
		}
		utils.PrintInfo("  1ctl domains check --domain %s", domain)
	}
	return nil
}

// --- Domain remove ------------------------------------------------------

func handleDomainsRemove(ctx context.Context, in domainsRemoveInput) error {
	domain, err := normalizeDomainArg(in.Domain)
	if err != nil {
		return err
	}
	appName := in.App

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
	if !utils.Confirm(fmt.Sprintf("Remove domain %s from app %s?", domain, appName), in.Yes) {
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

// --- Domain check -------------------------------------------------------

func handleDomainsCheck(ctx context.Context, in domainsCheckInput) error {
	domain, err := normalizeDomainArg(in.Domain)
	if err != nil {
		return err
	}

	ing, err := api.GetIngressByDomainName(domain)
	if err != nil {
		return printDetachedDomainStatus(domain, err)
	}

	status, err := api.GetDomainStatus(ing.IngressID.String(), domain, in.Probe)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to check domain %q: %s", domain, err.Error()), nil)
	}

	if utils.TryPrintJSON(status) {
		return nil
	}

	printDomainStatus(status)
	return nil
}

// --- Domain setup -------------------------------------------------------

func handleDomainsSetup(ctx context.Context, in domainsSetupInput) error {
	domain, err := normalizeDomainArg(in.Domain)
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

// --- Domain available ---------------------------------------------------

func handleDomainsAvailable(ctx context.Context, in domainsAvailableInput) error {
	if len(in.Domains) == 0 {
		return utils.NewError("at least one --domain is required", nil)
	}
	userID, orgID, err := domainAPIScope()
	if err != nil {
		return err
	}
	domains := make([]string, 0, len(in.Domains))
	for _, d := range in.Domains {
		domain, err := normalizeDomainArg(d)
		if err != nil {
			return err
		}
		domains = append(domains, domain)
	}
	results, err := api.CheckDomainAvailability(userID, orgID, api.DomainCheckRequest{Domains: domains, WithPrice: in.Price})
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to check availability: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(results) {
		return nil
	}
	rows := make([][]string, 0, len(results))
	for _, r := range results {
		price := ""
		if r.Price != nil {
			price = fmt.Sprintf("%.2f %s", r.Price.Amount, r.Price.Currency)
		}
		rows = append(rows, []string{r.Domain, boolText(r.Available), r.Status, price})
	}
	utils.PrintTable([]string{"DOMAIN", "AVAILABLE", "STATUS", "PRICE"}, rows)
	return nil
}

// --- Domain search ------------------------------------------------------

func handleDomainsSearch(ctx context.Context, in domainsSearchInput) error {
	if in.Name == "" {
		return utils.NewError("--name is required. Usage: 1ctl domains search --name <name>", nil)
	}
	userID, orgID, err := domainAPIScope()
	if err != nil {
		return err
	}
	results, err := api.SearchDomains(userID, orgID, api.DomainSearchRequest{
		DomainName: in.Name,
		Extensions: in.TLD,
		Period:     in.Period,
	})
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to search domains: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(results) {
		return nil
	}
	rows := make([][]string, 0, len(results))
	for _, r := range results {
		rows = append(rows, []string{
			r.Domain,
			boolText(r.Available),
			r.Status,
			fmt.Sprintf("%.2f %s", r.Price, r.Currency),
		})
	}
	utils.PrintTable([]string{"DOMAIN", "AVAILABLE", "STATUS", "PRICE"}, rows)
	return nil
}

// --- Managed domain handlers --------------------------------------------

func handleManagedDomainsList(ctx context.Context) error {
	userID, orgID, err := domainAPIScope()
	if err != nil {
		return err
	}
	domains, err := api.ListManagedDomains(userID, orgID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list managed domains: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(domains) {
		return nil
	}
	printManagedDomains(domains)
	return nil
}

func handleManagedDomainsAdd(ctx context.Context, in domainsManagedAddInput) error {
	if in.Domain == "" {
		return utils.NewError("--domain is required. Usage: 1ctl domains managed add --domain <domain>", nil)
	}
	userID, orgID, err := domainAPIScope()
	if err != nil {
		return err
	}
	domain, err := normalizeDomainArg(in.Domain)
	if err != nil {
		return err
	}
	created, ns, err := api.CreateManagedDomain(userID, orgID, api.DomainCreateRequest{Name: domain, IPAddress: in.IP})
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to add managed domain: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(map[string]interface{}{"domain": created, "nameserver_status": ns}) {
		return nil
	}
	utils.PrintSuccess("Managed domain %s added", created.Name)
	printManagedDomain(*created)
	printNameserverStatus(ns)
	return nil
}

func handleManagedDomainsVerify(ctx context.Context, in domainsManagedVerifyInput) error {
	if in.Domain == "" {
		return utils.NewError("--domain is required. Usage: 1ctl domains managed verify --domain <domain|domain-id>", nil)
	}
	userID, orgID, err := domainAPIScope()
	if err != nil {
		return err
	}
	domainID, err := resolveManagedDomainID(userID, orgID, in.Domain)
	if err != nil {
		return err
	}
	domain, ns, err := api.VerifyManagedDomain(userID, orgID, domainID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to verify managed domain: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(map[string]interface{}{"domain": domain, "nameserver_status": ns}) {
		return nil
	}
	printManagedDomain(*domain)
	printNameserverStatus(ns)
	return nil
}

func handleManagedDomainsDelete(ctx context.Context, in domainsManagedDeleteInput) error {
	if in.Domain == "" {
		return utils.NewError("--domain is required. Usage: 1ctl domains managed delete --domain <domain|domain-id>", nil)
	}
	userID, orgID, err := domainAPIScope()
	if err != nil {
		return err
	}
	domainID, err := resolveManagedDomainID(userID, orgID, in.Domain)
	if err != nil {
		return err
	}
	if !utils.Confirm(fmt.Sprintf("Delete managed domain %s?", in.Domain), in.Yes) {
		fmt.Println("Aborted.")
		return nil
	}
	if err := api.DeleteManagedDomain(userID, orgID, domainID); err != nil {
		return utils.NewError(fmt.Sprintf("failed to delete managed domain: %s", err.Error()), nil)
	}
	utils.PrintSuccess("Managed domain deleted")
	return nil
}

// --- DNS record handlers ------------------------------------------------

func handleDNSList(ctx context.Context, in dnsListInput) error {
	if in.Domain == "" {
		return utils.NewError("--domain is required. Usage: 1ctl domains dns list --domain <domain|domain-id>", nil)
	}
	userID, orgID, err := domainAPIScope()
	if err != nil {
		return err
	}
	domainID, err := resolveManagedDomainID(userID, orgID, in.Domain)
	if err != nil {
		return err
	}
	records, err := api.ListDNSRecords(userID, orgID, domainID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list DNS records: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(records) {
		return nil
	}
	printDNSRecords(records)
	return nil
}

func handleDNSCreate(ctx context.Context, in dnsCreateInput) error {
	if in.Domain == "" {
		return utils.NewError("--domain is required. Usage: 1ctl domains dns create --domain <domain|domain-id> --type A --name @ --data <value>", nil)
	}
	if in.Type == "" {
		return utils.NewError("--type is required. Usage: 1ctl domains dns create --domain <domain> --type A --name @ --data <value>", nil)
	}
	if in.Name == "" {
		return utils.NewError("--name is required. Usage: 1ctl domains dns create --domain <domain> --type A --name @ --data <value>", nil)
	}
	if in.Data == "" {
		return utils.NewError("--data is required. Usage: 1ctl domains dns create --domain <domain> --type A --name @ --data <value>", nil)
	}
	userID, orgID, err := domainAPIScope()
	if err != nil {
		return err
	}
	domainID, err := resolveManagedDomainID(userID, orgID, in.Domain)
	if err != nil {
		return err
	}
	record, err := api.CreateDNSRecord(userID, orgID, domainID, dnsCreateReqFromInput(in))
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to create DNS record: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(record) {
		return nil
	}
	utils.PrintSuccess("DNS record created")
	printDNSRecords([]api.DNSRecord{*record})
	return nil
}

func handleDNSUpdate(ctx context.Context, in dnsUpdateInput) error {
	userID, orgID, err := domainAPIScope()
	if err != nil {
		return err
	}
	domainID, err := resolveManagedDomainID(userID, orgID, in.Domain)
	if err != nil {
		return err
	}
	record, err := api.UpdateDNSRecord(userID, orgID, domainID, in.RecordID, dnsUpdateReqFromInput(in))
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to update DNS record: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(record) {
		return nil
	}
	utils.PrintSuccess("DNS record updated")
	printDNSRecords([]api.DNSRecord{*record})
	return nil
}

func handleDNSDelete(ctx context.Context, in dnsDeleteInput) error {
	userID, orgID, err := domainAPIScope()
	if err != nil {
		return err
	}
	domainID, err := resolveManagedDomainID(userID, orgID, in.Domain)
	if err != nil {
		return err
	}
	if !utils.Confirm(fmt.Sprintf("Delete DNS record %s?", in.RecordID), in.Yes) {
		fmt.Println("Aborted.")
		return nil
	}
	if err := api.DeleteDNSRecord(userID, orgID, domainID, in.RecordID); err != nil {
		return utils.NewError(fmt.Sprintf("failed to delete DNS record: %s", err.Error()), nil)
	}
	utils.PrintSuccess("DNS record deleted")
	return nil
}

// --- Domain purchase ----------------------------------------------------

func handleDomainsPurchase(ctx context.Context, in domainsPurchaseInput) error {
	if in.Domain == "" {
		return utils.NewError("--domain is required. Usage: 1ctl domains purchase --domain <domain>", nil)
	}
	userID, orgID, err := domainAPIScope()
	if err != nil {
		return err
	}
	domain, err := normalizeDomainArg(in.Domain)
	if err != nil {
		return err
	}
	resp, err := api.PurchaseDomain(userID, orgID, api.DomainPurchaseRequest{
		Domain: domain,
		Period: in.Period,
		Contact: &api.DomainContactInfo{
			FirstName:        in.FirstName,
			LastName:         in.LastName,
			Email:            in.Email,
			PhoneCountryCode: in.PhoneCountryCode,
			PhoneNumber:      in.PhoneNumber,
			Street:           in.Street,
			StreetNumber:     in.StreetNumber,
			PostalCode:       in.PostalCode,
			City:             in.City,
			State:            in.State,
			Country:          strings.ToUpper(in.Country),
			CompanyName:      in.Company,
		},
	})
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to start domain purchase: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(resp) {
		return nil
	}
	utils.PrintSuccess("Domain purchase checkout created")
	utils.PrintStatusLine("Domain", resp.Domain)
	utils.PrintStatusLine("Price", fmt.Sprintf("%.2f %s", resp.Price, resp.Currency))
	utils.PrintStatusLine("Intent ID", resp.IntentID)
	utils.PrintStatusLine("Checkout", resp.RedirectURL)
	return nil
}

func handleDomainsPurchaseStatus(ctx context.Context, in domainsPurchaseStatusInput) error {
	if in.IntentID == "" {
		return utils.NewError("--intent-id is required. Usage: 1ctl domains purchase-status --intent-id <intent-id>", nil)
	}
	userID, orgID, err := domainAPIScope()
	if err != nil {
		return err
	}
	status, err := api.GetDomainPurchaseStatus(userID, orgID, in.IntentID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get purchase status: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(status) {
		return nil
	}
	utils.PrintStatusLine("Intent ID", status.IntentID)
	utils.PrintStatusLine("Domain", status.Domain)
	utils.PrintStatusLine("Status", status.Status)
	return nil
}

// ========================================================================
// Shared helpers
// ========================================================================

type domainListEntry struct {
	DomainName string            `json:"domain_name"`
	AppLabel   string            `json:"app_label"`
	Namespace  string            `json:"namespace"`
	DnsConfig  api.DnsConfigType `json:"dns_config"`
	Kind       string            `json:"kind"`
	CreatedAt  string            `json:"created_at"`
}

func domainAPIScope() (userID, orgID string, err error) {
	userID = strings.TrimSpace(satuskyctx.GetUserID())
	orgID = strings.TrimSpace(satuskyctx.GetCurrentOrgID())
	if userID == "" {
		return "", "", utils.NewError("no current user ID is set. Run: 1ctl auth login", nil)
	}
	if orgID == "" {
		return "", "", utils.NewError("no current organization ID is set. Run: 1ctl org switch --id <org-id>", nil)
	}
	return userID, orgID, nil
}

func resolveManagedDomainID(userID, orgID, value string) (string, error) {
	if _, err := api.ParseUUID(value); err == nil {
		return value, nil
	}
	name, err := normalizeDomainArg(value)
	if err != nil {
		return "", err
	}
	domains, err := api.ListManagedDomains(userID, orgID)
	if err != nil {
		return "", utils.NewError(fmt.Sprintf("failed to resolve managed domain: %s", err.Error()), nil)
	}
	for _, domain := range domains {
		if strings.EqualFold(domain.Name, name) {
			return domain.DomainID, nil
		}
	}
	return "", utils.NewError(fmt.Sprintf("managed domain %q not found", value), nil)
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

func dnsCreateReqFromInput(in dnsCreateInput) api.DNSRecordCreateRequest {
	req := api.DNSRecordCreateRequest{
		Type: in.Type,
		Name: in.Name,
		Data: in.Data,
	}
	if in.TTL != 0 {
		v := in.TTL
		req.TTL = &v
	}
	if in.Priority != 0 {
		v := in.Priority
		req.Priority = &v
	}
	if in.Port != 0 {
		v := in.Port
		req.Port = &v
	}
	if in.Weight != 0 {
		v := in.Weight
		req.Weight = &v
	}
	if in.Flags != 0 {
		v := in.Flags
		req.Flags = &v
	}
	if in.Tag != "" {
		req.Tag = &in.Tag
	}
	return req
}

func dnsUpdateReqFromInput(in dnsUpdateInput) api.DNSRecordUpdateRequest {
	req := api.DNSRecordUpdateRequest{}
	if in.TypeSet {
		v := in.Type
		req.Type = &v
	}
	if in.NameSet {
		v := in.Name
		req.Name = &v
	}
	if in.DataSet {
		v := in.Data
		req.Data = &v
	}
	if in.TTLSet {
		v := in.TTL
		req.TTL = &v
	}
	if in.PrioritySet {
		v := in.Priority
		req.Priority = &v
	}
	if in.PortSet {
		v := in.Port
		req.Port = &v
	}
	if in.WeightSet {
		v := in.Weight
		req.Weight = &v
	}
	if in.FlagsSet {
		v := in.Flags
		req.Flags = &v
	}
	if in.TagSet {
		req.Tag = &in.Tag
	}
	return req
}

// --- Display helpers ----------------------------------------------------

func printManagedDomains(domains []api.Domain) {
	rows := make([][]string, 0, len(domains))
	for _, domain := range domains {
		rows = append(rows, []string{
			domain.Name,
			domain.DomainID,
			domain.Status,
			fmt.Sprintf("%d", domain.TTL),
			utils.FormatTimeAgo(domain.CreatedAt),
		})
	}
	utils.PrintTable([]string{"DOMAIN", "ID", "STATUS", "TTL", "CREATED"}, rows)
}

func printManagedDomain(domain api.Domain) {
	utils.PrintStatusLine("Domain", domain.Name)
	utils.PrintStatusLine("Domain ID", domain.DomainID)
	utils.PrintStatusLine("Status", domain.Status)
	utils.PrintStatusLine("TTL", fmt.Sprintf("%d", domain.TTL))
}

func printNameserverStatus(status *api.NameserverStatus) {
	if status == nil {
		return
	}
	utils.PrintStatusLine("Nameservers", boolText(status.Verified))
	if len(status.CurrentNameservers) > 0 {
		utils.PrintStatusLine("Current NS", strings.Join(status.CurrentNameservers, ", "))
	}
	if len(status.ExpectedNameservers) > 0 {
		utils.PrintStatusLine("Expected NS", strings.Join(status.ExpectedNameservers, ", "))
	}
	if status.Message != "" {
		utils.PrintStatusLine("Message", status.Message)
	}
}

func printDNSRecords(records []api.DNSRecord) {
	rows := make([][]string, 0, len(records))
	for _, r := range records {
		name := r.Name
		if name == "" {
			name = "@"
		}
		rows = append(rows, []string{r.RecordID, r.Type, name, r.Data, fmt.Sprintf("%d", r.TTL)})
	}
	utils.PrintTable([]string{"ID", "TYPE", "NAME", "DATA", "TTL"}, rows)
}

func boolText(v bool) string {
	if v {
		return "yes"
	}
	return "no"
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
	utils.PrintInfo("Attach it with: 1ctl domains add --domain %s --app <app>", domain)
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
	utils.PrintInfo("Attach it first with: 1ctl domains add --domain %s --app <app>", domain)
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
		utils.PrintInfo("Run setup details with: 1ctl domains setup --domain %s", status.DomainName)
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
	utils.PrintInfo("Next check: 1ctl domains check --domain %s --probe", status.DomainName)
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
