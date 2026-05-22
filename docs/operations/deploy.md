# Deployment

This page covers running the components on real hosts. Docker images and Ansible
playbooks for multi-host clusters land in later operations slices.

## Single host

Run `infra-api` with a file state backend for development:

```bash
infra-api
```

## Debian packages

Each component ships as a `.deb`. Build them locally:

```bash
make deb VERSION=1.0.0           # all components, amd64
ARCH=arm64 ./packaging/build-debs.sh 1.0.0 infra-api   # one component, arm64
```

Releases attach the `.deb` packages (amd64 and arm64) to each component's GitHub
Release alongside the raw binaries and checksums. Installing a service package
drops the binary in `/usr/local/bin` and a systemd unit under
`/lib/systemd/system`:

```bash
sudo dpkg -i infra-api_1.0.0_amd64.deb
sudo systemctl enable --now infra-api
```

## systemd

The long-running components ship systemd units under `deploy/systemd/`
(`infra-api`, `infra-controller-manager`, `infra-agent`, `infra-exporter`,
`infra-idp`). State lives in `/var/lib/infra` (provisioned via `StateDirectory`);
`infra-agent` runs with `CAP_NET_ADMIN` to manage bridges and iptables.

## Observability

`infra-exporter` exposes Prometheus metrics; a docker-compose stack with
Prometheus and Grafana dashboards is provided for local monitoring.

## Multi-host

For clusters, state moves to etcd and hosts are provisioned with Ansible. See
the architecture [overview](../architecture/overview.md).
