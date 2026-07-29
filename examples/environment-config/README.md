# Environment configuration guide example

This fixture uses a pinned, non-root nginx image to exercise Satusky
environment variables and secrets without building an image.

From the `1ctl` repository root:

```sh
cd examples/environment-config
1ctl deploy \
  --config ./satusky.toml \
  --wait \
  --wait-mode workload
```

The file supplies the non-secret `APP_ENV` value and declares
`GUIDE_ENV_TOKEN` as a required secret name. It never stores a secret value.

Delete the example when finished:

```sh
1ctl app delete guide-env-nginx --yes
```
