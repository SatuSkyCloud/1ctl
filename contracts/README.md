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

## Pinned backend endpoint contract

`backend-client-endpoints.json` is a vendored copy of
`architecture/client_endpoint_manifest.json` from the immutable backend commit
recorded in `backend-client-endpoints.provenance`. CI points the endpoint
compatibility test at this local copy so a missing sibling checkout cannot
silently skip the contract audit.

When the backend endpoint contract changes, deliberately refresh both files
from the reviewed backend commit and rerun the endpoint compatibility test.
