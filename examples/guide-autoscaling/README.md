# Autoscaling guide example

This project is the copy-paste fixture for the Satusky autoscaling guide. It
uses a pinned nginx image so testing HPA does not require a container build.

From the `1ctl` repository root:

```sh
cd examples/guide-autoscaling
1ctl deploy --config ./satusky.toml --wait --wait-mode workload
```

The HPA-enabled configuration sets a CPU request of `10m`, a CPU target of
`20%`, and a range of one to three replicas. Use `satusky.fixed.toml` to test
removal of the owned HPA:

```sh
1ctl deploy --config ./satusky.fixed.toml --wait --wait-mode workload
```

Delete the example deployment when finished:

```sh
1ctl app delete guide-autoscale-nginx --yes
```
