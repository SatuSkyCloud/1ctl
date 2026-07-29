# Node.js API example

This dependency-free Node.js HTTP API is the checked-in source for the
[Deploy a Node.js API](https://docs.satusky.com/guides/deploy-nodejs/) guide.

From this directory:

```sh
1ctl auth status
1ctl deploy --config satusky.toml --wait
1ctl app status guide-nodejs-api
APP_URL=$(1ctl -o json app get guide-nodejs-api | jq -er '.domain')
curl --fail --show-error --silent "$APP_URL/health"
1ctl app delete guide-nodejs-api --yes
```

The app binds to `0.0.0.0`, honors `PORT`, runs as the non-root `node` user,
and provides a strict `/health` endpoint for deployment verification.
