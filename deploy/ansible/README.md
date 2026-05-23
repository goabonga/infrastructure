# Multi-host test deployment (libvirt)

Provision a small libvirt/KVM cluster, install the Debian packages on it and
start the services. Use this to test the `.deb` artifacts on real machines.

## Prerequisites

On the libvirt host:

- `libvirtd`, `virt-install`, `qemu-img` and `cloud-image-utils` (`cloud-localds`)
- A Debian 12 generic cloud image at the path in `group_vars/all.yml`
  (`base_image`), for example:

  ```bash
  curl -L -o /var/lib/libvirt/images/debian-12-genericcloud-amd64.qcow2 \
    https://cloud.debian.org/images/cloud/bookworm/latest/debian-12-genericcloud-amd64.qcow2
  ```

- An SSH key at `~/.ssh/id_ed25519.pub` (or change `ssh_public_key`).

## Build the artifacts

From the repository root:

```bash
make deb VERSION=0.1.0   # .deb packages          -> dist/
make build-www           # dashboard (SPA embedded) -> build/infra-www
make build-provider      # Terraform provider       -> build/terraform-provider-infra
```

Keep `version` in `group_vars/all.yml` in sync with the `VERSION` you build.

## Deploy the full stack

Run as your normal user (not under sudo); `-K` provides the local become
password, and the playbook escalates on the VMs over SSH.

```bash
cd deploy/ansible

# 1. Create the VMs (control + two agents) on the libvirt default network.
ansible-playbook --ask-become-pass create-vms.yml

# 2. Deploy etcd, the control plane, the IdP, the dashboard, monitoring, agents.
ansible-playbook --ask-become-pass site.yml
```

`site.yml` first generates local credentials under `.credentials/` (gitignored):
the KMS key, the IdP ES256 keypair and a Terraform client secret. It then
deploys, on the control host:

| Component | Address | Notes |
| --- | --- | --- |
| etcd | `:2379` | shared state backend; every component uses it |
| infra-api | `:8080` | verifies IdP JWTs, secret/disk encryption enabled |
| infra-controller-manager | - | scheduler + reconcilers (leader-elected) |
| infra-exporter | `:9100` | Prometheus metrics |
| infra-idp | `:8081` | ES256 JWT issuer (client-credentials grant) |
| infra-www | `:8088` | dashboard (embedded SPA + API reverse proxy) |
| Prometheus | `:9090` | scrapes the exporter (native, no Docker) |
| Grafana | `:3000` | admin / infra; infra dashboard provisioned |

On each agent host it deploys `infra-agent` (sharing the etcd store, with the
KMS key and `GOA_NODE_ID`) and registers the host as a schedulable node.

## Provision infra with Terraform

With the stack up, run `terraform apply` from your terminal to build a topology
against it. See [terraform/README.md](terraform/README.md).

## Access

- Dashboard: `http://192.168.122.10:8088`
- Grafana:   `http://192.168.122.10:3000` (admin / infra)
- API:       `http://192.168.122.10:8080` (needs a Bearer JWT from the IdP)

## Tear down

```bash
ansible-playbook --ask-become-pass destroy-vms.yml
```

## Topology

Hosts and addresses are defined in `group_vars/all.yml` (`vms`) and mirrored in
`inventory.ini`. The default is one control host (`192.168.122.10`) and two
agents (`192.168.122.21`, `.22`) on the libvirt `default` NAT network.
