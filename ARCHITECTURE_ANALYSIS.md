# 1ctl — Architectural Issue Analysis & Fix Guide

## Root Cause Taxonomy

The 16 issues found in testing fall into 5 architectural patterns.
Only 2 are bugs in the traditional sense. The rest are pattern consistency gaps
that spread because no shared abstraction enforced the behavior.

---

## Pattern 1: Early Return Before JSON Output (11 handlers affected)

### The Problem

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

When `--output json` is set and the list is empty, the user gets "No things found"
instead of `[]`. This affects: **audit, cluster, credits, deploy list, domains,
environment, ingress, issuer, notifications, postgres, pricing, secret, service,
token, volumes** — 15 handlers total.

### Why It's Wrong

There are two output paths (JSON vs table) but only one is gated on the empty check.
JSON consumers expect an empty array `[]` — not a human-readable string.

### The Fix

A single helper that both paths converge on:

```go
// In internal/utils/output.go:

// PrintListOrJSON handles both JSON and table output for list commands.
// When --output json is set, prints items as JSON (empty or not).
// In table mode, prints emptyMsg if items is empty, then falls through
// so the caller can render the table.
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

Every list handler collapses to:

```go
items, _ := api.ListThings()
if utils.PrintListOrJSON(items, "No things found") {
    return nil
}
utils.PrintTable(headers, rows)
```

This eliminates the pattern duplication entirely. **11 handlers fixed in one function.**

---

## Pattern 2: Missing TryPrintJSON (4 handlers)

### The Problem

Some handlers have table output but `TryPrintJSON` was never added:
- `marketplace list`
- `user me`
- `user permissions` (empty case only — permissions array exists)
- `org list`

### Why It's Wrong

Phase 8 added `TryPrintJSON` to 8 commands but these 4 were missed.

### The Fix

Add `TryPrintJSON` call before any table output. Exact same pattern as Phase 8:

```go
if utils.TryPrintJSON(data) {
    return nil
}
```

Combined with the Pattern 1 fix (PrintListOrJSON), these handlers just need one line.

---

## Pattern 3: Missing Flags (--app, -y)

### The Problem

- `env list` and `env unset` don't accept `--app` flag (only `--deployment-id`)
- `notifications delete` doesn't support `-y` flag

### Why It's Wrong

Phase 2 added `--app` to deploy subcommands, volumes, logs, doctor — but `env list`/`env unset`
were missed. The resolveDeploymentID call passes `""` for appFlag:

```go
// env.go line ~64:
resolveDeploymentID(cmd.String("deployment-id"), "", cmd.String("config"))
//                                               ^^ should be cmd.String("app")
```

### The Fix

1. Add `--app` flag to env list and env unset subcommands
2. Change `""` to `cmd.String("app")` in resolveDeploymentID calls
3. Add `--yes/-y` flag to notifications delete subcommand

---

## Pattern 4: Output Format Inconsistency (2 handlers)

### 4a: Domain List JSON Uses FormatTimeAgo

```json
// domain list -o json currently:
{ "created_at": "just now", "updated_at": "3 minutes ago" }

// What it should be:
{ "created_at": "2026-06-17T19:42:30Z", "updated_at": "2026-06-17T19:43:00Z" }
```

**Root cause:** `api.FormatTimeAgo()` returns human-readable strings. It's called in
the handler before passing the data to `TryPrintJSON`, so the JSON output gets
the human-readable version. Other commands use `time.Time` in the struct and
let `json.Marshal` produce RFC3339.

**Fix:** Don't call `FormatTimeAgo` when building the data for JSON output.
Either use the raw `time.Time` fields, or have a JSON-specific struct that
omits the human-readable fields.

### 4b: Token List Exposes Full JWT in JSON

**Root cause:** The token struct includes the full JWT token value. Table mode
might truncate it, but JSON mode reveals the full token. This is a security
concern for CI/CD pipelines that capture JSON output.

**Fix:** Either:
- Redact the token field in JSON output (replace with `"***redacted***"`)
- OR remove the token field from the JSON struct entirely (separate struct for display vs API)

---

## Pattern 5: Pricing Backend Response Mismatch

### The Problem

`pricing list` (table mode) returns empty success. `pricing list -o json` returns
`"User not found"` error. This means the backend handler has two different code
paths for the same endpoint based on some request difference (probably Accept header
or query param).

### Why It's Wrong

The backend should return the same data regardless of format. The format is a
**presentation** concern, not an **authorization** concern.

### The Fix

Backend needs to unify the response path. The JSON serialization should not hit
a different handler or authorization check than the table serialization.

---

## Pattern 6: Minor Naming/UX (2 issues)

### 6a: `user info` Doesn't Exist

`user info` returns "No help topic for 'info'". The command is `user me`.
Adding `"info"` as an alias for `"me"` is a one-line fix.

### 6b: No Positional Arg Support for Deploy Status/Get

`deploy status <id>` fails. Users expect to pass IDs positionally.
Adding `Arguments: []cli.Argument{&cli.StringArgs{Name: "deployment-id", Min: 0, Max: 1}}`
enables this without breaking existing `--deployment-id <id>` usage.

---

## Summary: The Right Architectural Direction

The core problem is that **no shared abstraction enforces consistent output behavior**.
Every list handler independently implements its own version of "if empty print message,
if table render table, if JSON marshal JSON." This spreads bugs and makes sweeping
changes expensive.

The fix is one helper (`PrintListOrJSON`) that all 20+ list handlers call.
This is the same principle that made `resolveDeploymentID` work for 10+ commands:
**extract the pattern into a function, call it everywhere.**

| Fix | Effort | Files |
|---|---|---|
| `PrintListOrJSON` helper | 1 function | 1 |
| Adopt in all list handlers | ~20 one-line changes | ~15 |
| Add `TryPrintJSON` to marketplace, user, org | 3 one-line changes | 3 |
| Add `--app` to env list/unset | 4 lines | 1 |
| Add `-y` to notifications delete | 2 lines | 1 |
| Domain list JSON timestamps | Change FormatTimeAgo call site | 1 |
| Token JSON redaction | Add conditional in handler | 1 |
| `user me` alias | 1 line | 1 |
| Deploy status/get positional arg | 4 lines | 1 |
| Pricing backend | Backend fix | backend |
| **Total** | **~35 lines + 1 function** | **~20 files** |
