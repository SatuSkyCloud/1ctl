# 1ctl — Architectural Issue Analysis & Fix Guide

## Root Cause Taxonomy

The 16 issues found in testing fall into 5 architectural patterns.
Only 2 are bugs in the traditional sense. The rest are pattern consistency gaps
that spread because no shared abstraction enforced the behavior.

---

## Pattern 1: Early Return Before JSON Output (11 handlers affected)

Every list handler follows this structure:

```go
items, _ := api.ListThings()

if len(items) == 0 {
    utils.PrintInfo("No things found")   // ← human-readable message
    return nil                            // ← BYPASSES TryPrintJSON
}

if utils.TryPrintJSON(items) { return nil }  // ← JSON only reached when data exists
utils.PrintTable(headers, rows)
```

When `--output json` is set and the list is empty, the user gets a human-readable
message instead of `[]`.

### Commands That Produce Wrong Output

```
$ 1ctl issuer list -o json
💡 No certificate issuers found
   ^^^^^^^^^^^^^^^^^^^^^^^^^^^^  should be: []

$ 1ctl credits usage --days 1 -o json
💡 No machine usage found for the last 1 days
   ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^  should be: []

$ 1ctl secret list -o json
💡 No secrets found
   ^^^^^^^^^^^^^^^^^  should be: []

$ 1ctl env list --deployment-id ffe3e62f... -o json
💡 No environments found
   ^^^^^^^^^^^^^^^^^^^^^^  should be: []

$ 1ctl audit list -o json        (when empty — same issue)
$ 1ctl notifications list -o json (when empty — same issue)
$ 1ctl domains list -o json       (when empty — same issue)
$ 1ctl ingress list -o json       (when empty — same issue)
$ 1ctl service list -o json       (when empty — same issue)
$ 1ctl volumes list ... -o json   (when empty — same issue)
$ 1ctl postgres list -o json      (when empty — same issue)
$ 1ctl cluster zones -o json      (when empty — same issue)
```

**15 handlers total** — every command that has both `len(items) == 0 { PrintInfo; return }` and `TryPrintJSON`.

### The Fix

A single helper that both output paths converge on:

```go
// In internal/utils/output.go:

import "reflect"

// PrintListOrJSON handles both JSON and table output for list commands.
// When --output json is set, prints items as JSON (empty array or populated).
// In table mode, prints emptyMsg if items is empty and returns true.
// Returns false when the caller should render the table (items are non-empty,
// output is table mode).
func PrintListOrJSON(items interface{}, emptyMsg string) bool {
    if TryPrintJSON(items) {
        return true  // JSON handled, caller should return
    }
    // Table mode
    if reflect.ValueOf(items).Len() == 0 && emptyMsg != "" {
        PrintInfo(emptyMsg)
        return true  // Empty message printed, caller should return
    }
    return false  // Caller should render table
}
```

Every list handler collapses from this:

```go
// BEFORE (broken — two output paths that never meet):
items, _ := api.ListThings()
if len(items) == 0 {
    utils.PrintInfo("No things found")
    return nil
}
if utils.TryPrintJSON(items) { return nil }
utils.PrintTable(headers, rows)

// AFTER (fixed — single convergence point):
items, _ := api.ListThings()
if utils.PrintListOrJSON(items, "No things found") {
    return nil
}
utils.PrintTable(headers, rows)
```

**11 handlers fixed with one function.** The pattern eliminated: empty-check-then-
human-message-then-return that bypasses JSON output entirely.

---

## Pattern 2: Missing TryPrintJSON Entirely (4 handlers)

Some handlers have table-formatted output but `TryPrintJSON` was never added.
Phase 8 missed them. The `-o json` flag is silently ignored — output is always
the human-readable table format.

### Commands That Produce Wrong Output

```
$ 1ctl marketplace list -o json
Marketplace Apps
────────────────
ID: 1c6da9af-04ed-440a-8088-af47367b50f1
Name: uptime-kuma
Category:
Description: A fancy self-hosted monitoring tool...
            ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^  table output despite -o json

$ 1ctl user me -o json
User Profile
────────────
ID: e8dfc6df-0eb5-4c13-ae92-10915d21f6f0
Email: mingerz.k@gmail.com
Name: mingerzk
            ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^  table output despite -o json

$ 1ctl user permissions -o json
User Permissions
────────────────
  audit:export    Export audit logs to CSV/JSON format
  audit:read      View audit logs and activity history
            ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^  table output despite -o json
```

### The Fix

Add `TryPrintJSON` before any table/friendly output. Exact same pattern as Phase 8:

```go
if utils.TryPrintJSON(data) {
    return nil
}
```

Combined with Pattern 1's `PrintListOrJSON` helper, these become:

```go
// marketplace list:
if utils.PrintListOrJSON(apps, "No marketplace apps available") { return nil }
// ... render table ...

// user me:
if utils.TryPrintJSON(user) { return nil }
// ... render profile ...

// user permissions:
if utils.TryPrintJSON(permissions) { return nil }
// ... render permissions ...
```

---

## Pattern 3: Missing Flags (3 commands)

### 3a: `env list` and `env unset` — Missing `--app`

Phase 2 added `--app` to deploy subcommands, volumes, logs, doctor, secret — but
`env list` and `env unset` were missed. The `resolveDeploymentID` call passes `""`.

```
$ 1ctl env list --app fullstack-api
Incorrect Usage: flag provided but not defined: -app
                 ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^  flag doesn't exist

$ 1ctl env list --help
OPTIONS:
   --deployment-id string  Filter by deployment ID
                           ^^^^^^^^^^  only --deployment-id, no --app
```

**Root cause in env.go:**
```go
// env list handler:
resolveDeploymentID(cmd.String("deployment-id"), "", cmd.String("config"))
//                                               ^^ should be cmd.String("app")
```

**Fix:**
1. Add `&cli.StringFlag{Name: "app", Usage: "App name to resolve..."}` to env list + env unset flags
2. Change `""` to `cmd.String("app")` in both `resolveDeploymentID` calls

### 3b: `notifications delete` — Missing `-y`

```
$ 1ctl notifications delete --id abc-123 -y
Incorrect Usage: flag provided but not defined: -y
                 ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^  flag doesn't exist

$ 1ctl notifications delete --help
OPTIONS:
   --id string  Notification ID to delete
                ^^^^^^^^^^  no --yes/-y flag
```

**Fix:** Add `&cli.BoolFlag{Name: "yes", Aliases: []string{"y"}, Usage: "Skip confirmation prompt"}`.

Note: Currently `notifications delete` deletes immediately without confirmation.
Adding `-y` would be a no-op right now, but it enables adding a confirmation
prompt later without breaking existing scripts.

---

## Pattern 4: Output Format Inconsistency (2 handlers)

### 4a: Domain List JSON Uses Human-Readable Time

When domains exist, the JSON `created_at` field contains `"just now"` or `"3 minutes ago"`
instead of an RFC3339 timestamp like every other command:

```json
// What domain list -o json produces:
[
  {
    "domain_name": "thirstygiraffe-lpdtnap.satusky.com",
    "app_label": "fullstack-api",
    "created_at": "just now",
                  ^^^^^^^^^^^^  human-readable, not machine-parseable
  }
]

// What every other command produces:
{
  "created_at": "2026-06-17T19:42:30Z",
                ^^^^^^^^^^^^^^^^^^^^^^^^  RFC3339, parseable by jq/scripts
}
```

**Root cause:** The handler calls `api.FormatTimeAgo()` on the `CreatedAt` field
before passing the struct to `TryPrintJSON`. `FormatTimeAgo` replaces the
`time.Time` with a string like "just now". Other commands keep the raw `time.Time`
and let `json.Marshal` produce RFC3339.

**Fix:** Use the raw `time.Time` field in the JSON struct. `FormatTimeAgo` should
only be called when building table rows, not when the struct is passed to JSON.

### 4b: Token List Exposes Full JWT

```
$ 1ctl token list -o json
[
  {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3ODg5NjIyNTUs...",
             ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
             Full JWT token exposed in JSON output — security concern
  }
]
```

**Root cause:** The token struct includes the full JWT token value. Table mode
doesn't display it, but JSON mode reveals it. This is a security risk for CI/CD
pipelines that capture `--output json` in logs or artifacts.

**Fix:** Redact the token field in JSON output. Options:
- Replace with `"***redacted***"` in the handler before passing to `TryPrintJSON`
- Or remove the field from the JSON struct (separate display struct vs API struct)

---

## Pattern 5: Pricing Backend Response Mismatch

```
$ 1ctl pricing list
💡 No pricing configurations found
   ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^  table mode: empty success

$ 1ctl pricing list -o json
💡 No pricing configurations found
   ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^  JSON mode: same output (was "User not found"
                                     in earlier test run — intermittent backend issue)
```

**Root cause:** The pricing endpoint may return different responses based on
request format (Accept header or query parameter). The JSON path should return
the same data as the table path — format is a presentation concern, not an
authorization or data concern.

**Fix:** Backend needs to unify the response path. If no pricing configs exist,
both modes should return an empty list.

---

## Pattern 6: Naming / UX (2 issues)

### 6a: `user info` Doesn't Exist

```
$ 1ctl user info
No help topic for 'info'
   ^^^^^^^^^^^^^^^^^^^^^^^^  subcommand doesn't exist, no helpful error

$ 1ctl user --help
COMMANDS:
   me           Show user profile
   update       Update user profile
   ...
```

**Fix:** Add `"info"` as an alias for `"me"`:
```go
Name:    "me",
Aliases: []string{"info"},
```

### 6b: No Positional Arg Support for Deploy Status/Get

```
$ 1ctl deploy status ffe3e62f-c5ae-4ea1-b012-7805e4d21abe
❌ no --deployment-id and no satusky.toml found
Run '1ctl deploy' in your project directory or pass --deployment-id
   ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
   UUID was silently ignored — treated as unrecognized input

$ 1ctl deploy status --deployment-id ffe3e62f-c5ae-4ea1-b012-7805e4d21abe
❌ failed to get deployment status: Deployment not found
   ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
   Actually tried to look up the ID (correct behavior, deployment was deleted)
```

**Root cause:** `deploy status` and `deploy get` have no `Arguments` defined.
A bare UUID on the command line is silently dropped instead of being treated as
a shorthand for `--deployment-id`.

**Fix:** Add positional arg support:
```go
Arguments: []cli.Argument{
    &cli.StringArgs{Name: "deployment-id", Min: 0, Max: 1},
},
```
Then in the handler, fall back from flag to positional:
```go
depID := cmd.String("deployment-id")
if depID == "" {
    args := cmd.StringArgs("deployment-id")
    if len(args) > 0 { depID = args[0] }
}
```

---

## Summary: The Right Architectural Direction

The core problem is that **no shared abstraction enforces consistent output behavior**.
Every list handler independently implements its own version of "if empty print message,
if table render table, if JSON marshal JSON." This spreads bugs and makes sweeping
changes expensive.

The fix is one helper (`PrintListOrJSON`) that all 20+ list handlers call.
This is the same principle that made `resolveDeploymentID` work for 10+ commands:
**extract the pattern into a function, call it everywhere.**

| Pattern | Commands Affected | Fix |
|---|---|---|
| 1. Early return before JSON | 15 handlers | `PrintListOrJSON()` helper + adopt everywhere |
| 2. Missing TryPrintJSON | 3 handlers | Add 1-line `TryPrintJSON` call |
| 3. Missing flags | 3 commands | Add `--app` / `-y` flags |
| 4. Output format mismatch | 2 handlers | Separate display vs JSON serialization |
| 5. Backend response mismatch | 1 handler | Backend fix |
| 6. Naming / UX | 2 commands | Add alias + positional arg |

| Metric | Value |
|---|---|
| Total files changed | ~20 (15 handlers + 1 helper + flags + aliases) |
| Net new lines | ~35 lines + 1 helper function |
| Handlers fixed by helper | 15 (one-line change each) |
| Backend changes needed | 1 (pricing endpoint) |
