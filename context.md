# Code Context: 1ctl Domain Management CLI

## Commits Scoped
- `8d30b44` feat: attach custom domains as aliases
- `16ac0c1` feat: expose managed domain dns commands

## Files Changed (git diff HEAD~2..HEAD)

| File | Status | Lines | Purpose |
|------|--------|-------|---------|
| `internal/api/domains.go` | **new** | +136 | API client functions for domain registry, managed DNS, DNS records |
| `internal/api/client.go` | modified | +27 | Added `ListDomainAliases`, `AttachDomain`, `DetachDomain` |
| `internal/api/models.go` | modified | +23 | Added `IngressAlias`, `AttachDomainRequest`, `DetachDomainRequest` types |
| `internal/commands/domain_registry.go` | **new** | +648 | All new domain subcommands: available, search, managed, dns, purchase, purchase-status |
| `internal/commands/domains.go` | modified | +173/-18 | Refactored `handleDomainsAdd` (alias code path), `handleDomainsRemove` (detach), `handleDomainsList` (show aliases), added `normalizeDomainArg`, `currentOrgUUID` |
| `internal/commands/domains_test.go` | modified | +74 | New tests: `TestNormalizeDomainArg`, expanded subcommand structure checks, helper functions |

---

## 1. Complete Domain Command Tree

All commands hang off `commands.DomainsCommand()` registered in `cmd/1ctl/main.go:79`.

```
domains (alias: domain)
├── list                          — list all domains (primary + custom aliases)
├── add <domain>                  — add custom domain to app
│   Flags: --app (required), --port (default 8080), --custom-dns, --with-www
├── remove <domain>               — remove domain (now uses DetachDomain)
│   Flags: --app (required), --yes/-y
├── check <domain>                — check backend/route/DNS/TLS/HTTP status
│   Flags: --probe
├── setup <domain>                — show DNS setup instructions
├── available <domain> [domain…]  — check registration availability
│   Flags: --price
├── search <name>                 — search domains across TLDs
│   Flags: --tld (repeatable), --period
├── managed
│   ├── list                      — list SatuSky-managed domains
│   ├── add <domain>              — add domain zone to SatuSky DNS
│   │   Flags: --ip
│   ├── verify <domain|id>        — verify nameservers
│   └── delete <domain|id>        — delete domain zone
│       Flags: --yes/-y
├── dns
│   ├── list <domain|id>          — list DNS records
│   ├── create <domain|id>        — create DNS record
│   │   Flags: --type (required), --name (required), --data (required),
│   │           --ttl, --priority, --port, --weight, --flags, --tag
│   ├── update <domain|id> <record-id>  — update DNS record
│   │   Flags: --type, --name, --data, --ttl, --priority, --port, --weight, --flags, --tag
│   └── delete <domain|id> <record-id>  — delete DNS record
│       Flags: --yes/-y
├── purchase <domain>             — start domain purchase checkout
│   Flags: --period, --first-name, --last-name, --email, --phone-country-code,
│          --phone-number, --street, --street-number, --postal-code, --city,
│          --state, --country, --company
└── purchase-status <intent-id>   — check purchase intent status
```

---

## 2. OpenProvider Integration

The CLI **does not** directly integrate OpenProvider. It is a thin API client delegating to the backend.

**Backend API routes** (`internal/api/domains.go`):

| Function | Backend Route | Method |
|----------|--------------|--------|
| `CheckDomainAvailability` | `/domains/check/:userID/:orgID` | POST |
| `SearchDomains` | `/domains/search/:userID/:orgID` | POST |
| `PurchaseDomain` | `/domains/purchase-intent/:userID/:orgID` | POST |
| `GetDomainPurchaseStatus` | `/domains/purchase-intent/:userID/:orgID/:intentID` | GET |
| `ListManagedDomains` | `/domains/list/:userID/:orgID` | GET |
| `CreateManagedDomain` | `/domains/create/:userID/:orgID` | POST |
| `DeleteManagedDomain` | `/domains/delete/:userID/:orgID/:domainID` | DELETE |
| `VerifyManagedDomain` | `/domains/verify/:userID/:orgID/:domainID` | GET |
| `ListDNSRecords` | `/domains/:userID/:orgID/:domainID/records` | GET |
| `CreateDNSRecord` | `/domains/:userID/:orgID/:domainID/records` | POST |
| `UpdateDNSRecord` | `/domains/:userID/:orgID/:domainID/records/:recordID` | PUT |
| `DeleteDNSRecord` | `/domains/:userID/:orgID/:domainID/records/:recordID` | DELETE |

**RELEASE_NOTES.md:531** confirms: *"Full lifecycle for external domain registration and DNS via OpenProvider"* — this happens server-side.

### Key types for OpenProvider-related flows (`internal/api/models.go`):

```go
// models.go:500-509
type Domain struct {
    DomainID  string    `json:"domain_id"`
    Name      string    `json:"name"`
    Status    string    `json:"status"`
    TTL       int       `json:"ttl"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

// models.go:533-543
type DomainAvailabilityResult struct {
    Domain       string       `json:"domain"`
    Available    bool         `json:"available"`
    Status       string       `json:"status"`
    Reason       string       `json:"reason,omitempty"`
    IsPremium    bool         `json:"is_premium,omitempty"`
    Price        *DomainPrice `json:"price,omitempty"`
    PremiumPrice float64      `json:"premium_price,omitempty"`
}

// models.go:544-555
type DomainSearchResult struct {
    Domain    string  `json:"domain"`
    Extension string  `json:"extension"`
    Available bool    `json:"available"`
    Status    string  `json:"status"`
    Price     float64 `json:"price"`
    Currency  string  `json:"currency"`
    IsPremium bool    `json:"is_premium"`
    Period    int     `json:"period"`
}

// models.go:556-562
type NameserverStatus struct {
    Verified            bool     `json:"verified"`
    CurrentNameservers  []string `json:"current_nameservers"`
    ExpectedNameservers []string `json:"expected_nameservers"`
    Message             string   `json:"message"`
}

// models.go:564-578
type DomainContactInfo struct {
    FirstName        string `json:"first_name"`
    LastName         string `json:"last_name"`
    Email            string `json:"email"`
    PhoneCountryCode string `json:"phone_country_code"`
    PhoneNumber      string `json:"phone_number"`
    Street           string `json:"street"`
    StreetNumber     string `json:"street_number"`
    PostalCode       string `json:"postal_code"`
    City             string `json:"city"`
    State            string `json:"state,omitempty"`
    Country          string `json:"country"`
    CompanyName      string `json:"company_name,omitempty"`
}

// models.go:606-614
type DomainPurchaseIntentResponse struct {
    SessionID   string  `json:"session_id"`
    RedirectURL string  `json:"redirect_url"`
    Domain      string  `json:"domain"`
    Price       float64 `json:"price"`
    Currency    string  `json:"currency"`
    IntentID    string  `json:"intent_id"`
}
```

---

## 3. Hostname Validation and Wildcard Rejection

### Layer 1: `normalizeDomainArg()` in `internal/commands/domains.go:533-545`

This is the CLI-level pre-processor applied to **every** domain argument. It runs before calling the validator or API.

```go
func normalizeDomainArg(domain string) (string, error) {
    domain = strings.ToLower(strings.TrimSpace(domain))
    domain = strings.TrimSuffix(domain, ".")
    // Reject URLs
    if strings.HasPrefix(domain, "http://") || strings.HasPrefix(domain, "https://") || strings.Contains(domain, "/") {
        return "", utils.NewError("domain must be a hostname, not a URL", nil)
    }
    // Reject wildcards with a clear reason
    if strings.HasPrefix(domain, "*.") {
        return "", utils.NewError("wildcard domains are not supported yet because SatuSky custom-domain TLS currently uses HTTP-01 validation", nil)
    }
    // Delegate to validator.ValidateDomain for structural validation
    if err := validator.ValidateDomain(domain); err != nil {
        return "", err
    }
    return domain, nil
}
```

**Key point**: The CLI **rejects** wildcards at this layer with an explicit error message. Even though `validator.ValidateDomain` accepts `*.example.com` (it was built for the deploy command's `--domain` flag where wildcards might be acceptable in different contexts), the `normalizeDomainArg` gate blocks them before they reach the validator.

### Layer 2: `validator.ValidateDomain()` in `internal/validator/resource.go:85-96`

```go
var domainPattern = regexp.MustCompile(`^(\*\.)?[a-zA-Z0-9][-a-zA-Z0-9]*(\.[a-zA-Z0-9][-a-zA-Z0-9]*)*\.[a-zA-Z]{2,}$`)

func ValidateDomain(domain string) error {
    if domain == "" {
        return nil  // empty is allowed (optional field)
    }
    if !domainPattern.MatchString(domain) {
        return utils.NewError("invalid domain format", nil)
    }
    return nil
}
```

The regex accepts:
- `example.com`
- `sub.example.com`
- `*.example.com` (wildcards)
- Empty string (no-op)
- Rejects: no TLD, invalid characters, missing dots

**Note on inconsistency**: The validator allows wildcards (`*.example.com`) but `normalizeDomainArg` blocks them. This means wildcards can be used via `deploy --domain` (which calls `validator.ValidateDomain` directly at `deploy.go:625`) but are blocked for the `domains` command tree. This is intentional: the `domains` commands use HTTP-01 TLS validation which doesn't support wildcard certs.

### Call sites:
- `internal/commands/domains.go` — `handleDomainsAdd:201`, `handleDomainsRemove:325`, `handleDomainsCheck:361`, `handleDomainsSetup:388`
- `internal/commands/domain_registry.go` — `handleDomainsAvailable:194`, `handleManagedDomainsAdd:277`, `handleDomainsPurchase:465`, `resolveManagedDomainID:530`
- `internal/commands/deploy.go:625` — `deploy --domain` validation (uses `validator.ValidateDomain` directly, no `normalizeDomainArg`)

---

## 4. Custom Domain Alias Architecture

### New types (`internal/api/models.go:193-218`)

```go
type IngressAlias struct {
    AliasID    uuid.UUID     `json:"alias_id"`
    IngressID  uuid.UUID     `json:"ingress_id"`
    DomainName string        `json:"domain_name"`
    DnsConfig  DnsConfigType `json:"dns_config"`
    DomainID   *uuid.UUID    `json:"domain_id,omitempty"`
    IsRedirect bool          `json:"is_redirect"`
    RedirectTo string        `json:"redirect_to,omitempty"`
    CreatedAt  time.Time     `json:"created_at"`
    UpdatedAt  time.Time     `json:"updated_at"`
}

type AttachDomainRequest struct {
    OrgID           uuid.UUID `json:"org_id"`
    DomainName      string    `json:"domain_name"`
    WithWWWRedirect bool      `json:"with_www_redirect"`
}

type DetachDomainRequest struct {
    OrgID      uuid.UUID `json:"org_id"`
    DomainName string    `json:"domain_name"`
}
```

### API functions (`internal/api/client.go:747-771`)

```go
func ListDomainAliases(ingressID string) ([]IngressAlias, error)
func AttachDomain(ingressID string, req AttachDomainRequest) (*IngressAlias, error)
func DetachDomain(ingressID string, req DetachDomainRequest) error
```

### Flow in `handleDomainsAdd` (`internal/commands/domains.go:201-291`)

1. Normalize and validate domain via `normalizeDomainArg()`
2. Resolve deployment by app label
3. Find matching service for deployment
4. **If custom domain** (`!isSatuskyHost` or `--custom-dns`):
   - Get current org UUID via `currentOrgUUID()`
   - If no existing ingress, create a default platform ingress first (auto-generated domain via `api.GenerateDomainName`)
   - Call `api.AttachDomain()` to attach the custom domain as an alias
   - Print DNS setup instructions from `api.GetDomainStatus()`
5. **If platform domain** (legacy path): call `api.UpsertIngress()` directly

### Flow in `handleDomainsRemove` (`internal/commands/domains.go:322-350`)

1. Normalize domain
2. Find existing ingress by domain name
3. Cross-app safety check (refuse if domain belongs to different app)
4. Confirm with user
5. Call `api.DetachDomain()` instead of `api.DeleteIngress()`

### Flow in `handleDomainsList` (`internal/commands/domains.go:140-194`)

- Lists ingresses and enriches with aliases via `api.ListDomainAliases()`
- Excludes `IsRedirect` aliases from display
- Shows `kind: "primary"` for platform domains, `kind: "custom"` for aliases

---

## 5. DNS Record Management

### Types (`internal/api/models.go:510-523, 623-648`)

```go
type DNSRecord struct {
    RecordID  string    `json:"record_id"`
    DomainID  string    `json:"domain_id"`
    Type      string    `json:"type"`
    Name      string    `json:"name"`
    Data      string    `json:"data"`
    Priority  *int      `json:"priority,omitempty"`
    Port      *int      `json:"port,omitempty"`
    Weight    *int      `json:"weight,omitempty"`
    TTL       int       `json:"ttl"`
    Flags     *int      `json:"flags,omitempty"`
    Tag       *string   `json:"tag,omitempty"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

type DNSRecordCreateRequest struct {
    Type     string  `json:"type"`
    Name     string  `json:"name"`
    Data     string  `json:"data"`
    TTL      *int    `json:"ttl,omitempty"`
    Priority *int    `json:"priority,omitempty"`
    Port     *int    `json:"port,omitempty"`
    Weight   *int    `json:"weight,omitempty"`
    Flags    *int    `json:"flags,omitempty"`
    Tag      *string `json:"tag,omitempty"`
}

type DNSRecordUpdateRequest struct {
    Type     *string `json:"type,omitempty"`  // all pointer fields = partial update
    Name     *string `json:"name,omitempty"`
    // ... same optional fields ...
}
```

Supported record types: A, AAAA, CNAME, MX, TXT, NS, SRV, CAA

### Helper: `resolveManagedDomainID()` (`internal/commands/domain_registry.go:517-535`)

Accepts either a domain name or a UUID:
- If value is a valid UUID → return as-is
- Otherwise normalize as domain name, fetch all managed domains, match by case-insensitive name

### Auth scope: `domainAPIScope()` (`domain_registry.go:511-516`)

Returns `userID` and `orgID` from context. Returns clear error messages if not set.

---

## 6. Test Files for Domain Commands

### `internal/commands/domains_test.go` (full file, 135 lines)

Tests:
1. **`TestDomainsCommandStructure`** — validates command tree:
   - Top-level command name is `"domains"` with alias `"domain"`
   - All 11 subcommand names present: `list`, `add`, `remove`, `check`, `setup`, `available`, `search`, `managed`, `dns`, `purchase`, `purchase-status`
   - `add` has `--with-www` flag
   - `managed` has subcommands: `list`, `add`, `verify`, `delete`
   - `dns` has subcommands: `list`, `create`, `update`, `delete`

2. **`TestNormalizeDomainArg`** — table-driven test covering:
   - Lowercase + trim: `" App.Example.COM. "` → `"app.example.com"`
   - Reject URL: `"https://example.com"` → error
   - Reject wildcard: `"*.example.com"` → error
   - Reject path: `"example.com/path"` → error
   - Reject invalid: `"not a domain"` → error

3. **`TestSurfaceReframe_HiddenCommands`** — verifies `service`, `ingress`, `issuer` are hidden

4. Helper functions: `containsString`, `subNames`, `findSubcommand`, `hasFlag`

### `internal/validator/resource_test.go` — `TestValidateDomain` (lines 54-76)

Tests the validator regex (accepts wildcards, empty, subdomains; rejects invalid chars, missing TLD):
```go
{"valid domain", "example.com", false},
{"valid subdomain", "sub.example.com", false},
{"valid wildcard", "*.example.com", false},
{"valid multi-level", "a.b.example.com", false},
{"empty domain", "", false},
{"invalid format", "invalid", true},
{"invalid chars", "test!.com", true},
{"missing tld", "example", true},
```

---

## 7. Architecture Summary

```
cmd/1ctl/main.go
  └─ commands.DomainsCommand()          [domains.go:30]
       ├── domains.go                    — list/add/remove/check/setup (app-oriented)
       │   ├── normalizeDomainArg()      — pre-validation (rejects wildcards, URLs)
       │   └── currentOrgUUID()          — org context helper
       └── domain_registry.go            — available/search/managed/dns/purchase/purchase-status
           ├── domainAPIScope()          — userID+orgID from context
           ├── resolveManagedDomainID()  — name→UUID resolver
           ├── dnsCreateRequestFromFlags()
           ├── dnsUpdateRequestFromFlags()
           └── printer helpers: printManagedDomains, printNameserverStatus, printDNSRecords

internal/api/
  ├── client.go         — GetIngressByDomainName, ListDomainAliases, AttachDomain, DetachDomain, GetDomainStatus
  ├── domains.go        — all managed domain & DNS record API calls (12 functions)
  ├── models.go         — Domain, DNSRecord, IngressAlias, Attach/DetachDomainRequest, NameserverStatus, prices, purchase types
  └── utils.go          — GenerateDomainName, ParseUUID, SafeInt32

internal/validator/
  └── resource.go       — ValidateDomain() regex: ^(\*\.)?[a-zA-Z0-9][-a-zA-Z0-9]*(\.[a-zA-Z0-9][-a-zA-Z0-9]*)*\.[a-zA-Z]{2,}$
```

## 8. Key Design Decisions / Risks

1. **Wildcard inconsistency**: `validator.ValidateDomain` accepts `*.example.com` but `normalizeDomainArg` blocks it. The validator was built for `deploy --domain` which may have different TLS requirements.

2. **No local OpenProvider SDK**: All domain registration/search/purchase goes through the SatuSky backend API. The CLI is purely a relay.

3. **Custom domain aliases require an existing platform ingress**: If an app doesn't have a default ingress yet, the CLI auto-creates one before attaching a custom domain. This chaining happens in `handleDomainsAdd` (lines 259-278).

4. **Domain removal now detaches instead of deleting**: `handleDomainsRemove` switched from `api.DeleteIngress()` to `api.DetachDomain()`. This means removing a custom domain no longer tears down the entire ingress — just the alias.

5. **`--with-www` flag**: Supported by `AttachDomainRequest.WithWWWRedirect`. The backend handles the actual www redirect configuration.

6. **Purchase flow returns a redirect URL**: The `purchase` command returns a `DomainPurchaseIntentResponse` with a `RedirectURL` for completing checkout in a browser.

---

## 9. Start Here

Open **`internal/commands/domains.go`** first — it contains the root `DomainsCommand()`, the alias attachment/detachment logic, `normalizeDomainArg`, and `currentOrgUUID`. Then read **`internal/commands/domain_registry.go`** for all the new registry/managed/dns/purchase subcommands. Finally, reference **`internal/api/domains.go`** and **`internal/api/models.go`** for the backend API contract.
