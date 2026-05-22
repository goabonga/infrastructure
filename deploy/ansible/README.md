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

## Build the packages

From the repository root, build the `.deb` files into `dist/`:

```bash
make deb VERSION=0.1.0
```

Keep `version` in `group_vars/all.yml` in sync with the `VERSION` you build.

## Provision and deploy

```bash
cd deploy/ansible

# 1. Create the VMs (control + two agents) on the libvirt default network.
ansible-playbook create-vms.yml

# 2. Install the packages and start the services.
ansible-playbook site.yml
```

`site.yml` installs the control-plane packages on the `control` host
(`infra-api`, `infra-controller-manager`, `infra-idp`, `infra-exporter`) and the
agent packages on the `agents` hosts, then waits for the API health check.

## Tear down

```bash
ansible-playbook destroy-vms.yml
```

## Topology

Hosts and addresses are defined in `group_vars/all.yml` (`vms`) and mirrored in
`inventory.ini`. The default is one control host (`192.168.122.10`) and two
agents (`192.168.122.21`, `.22`) on the libvirt `default` NAT network.
