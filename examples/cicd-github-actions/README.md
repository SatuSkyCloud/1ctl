# CI/CD GitHub Actions example

This directory is the complete project used by the
[CI/CD Integration guide](https://docs.satusky.com/guides/cicd/). Its workflow
builds an immutable `1ctl` source revision whose commands match the guide,
deploys this Go health service, verifies the canonical status and public route,
and rolls back only when post-deploy verification fails.

## Run locally

```bash
go test ./...
go run .
curl --fail http://localhost:8080/health
```

## Deploy from GitHub

1. Copy this directory to the root of a GitHub repository.
2. Open **Settings → Environments → New environment**, enter `production`,
   select **Configure environment**, and add the required deployment protection
   rules.
3. Under **Environment secrets**, select **Add environment secret**, use the
   name `SATUSKY_API_KEY`, and paste the dedicated token value.
4. For a non-production control plane only, add an Environment variable named
   `SATUSKY_API_URL` under **Environment variables**.
5. Push to `main`.

The workflow is
`.github/workflows/deploy-production.yml`. Update `CLI_COMMIT` only after
validating that revision's generated `contracts/cli.json` and tests.
