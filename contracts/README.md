# Generated contracts

These files are generated from the same Go sources used to build `1ctl`:

- `cli.json` describes the registered command tree, documented positional
  arguments, flags, aliases, defaults, and hidden status.
- `api-types.ts` contains TypeScript representations of exported API types.

Regenerate them after changing commands or API models:

```sh
go run ./cmd/contractgen
```

CI runs `go run ./cmd/contractgen --check` and rejects stale artifacts.
Consumers should use a pinned release artifact for reproducible builds, or
explicitly checkout `main` when they intentionally track the latest contract.
