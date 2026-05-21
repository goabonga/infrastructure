# Deployment

This page covers running the components on real hosts. The deployment tooling
(systemd units, `.deb` packages, Docker images and Ansible playbooks for
multi-host clusters) lands with the operations milestone; this page tracks the
intended shape.

## Single host

Run `infra-api` with a file state backend for development:

```bash
infra-api
```

## systemd

Each long-running component ships a systemd unit (`infra-api`,
`infra-controller-manager`, `infra-agent`, `infra-exporter`, `infra-idp`).

## Observability

`infra-exporter` exposes Prometheus metrics; a docker-compose stack with
Prometheus and Grafana dashboards is provided for local monitoring.

## Multi-host

For clusters, state moves to etcd and hosts are provisioned with Ansible. See
the architecture [overview](../architecture/overview.md).
