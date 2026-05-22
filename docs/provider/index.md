# Terraform provider

`terraform-provider-infra` manages infrastructure resources through the
`infra-api` control plane, so the same resource model is available as code.

```hcl
terraform {
  required_providers {
    infra = {
      source = "goabonga/infra"
    }
  }
}

provider "infra" {
  endpoint = "http://[::1]:8080"
}

resource "infra_vpc" "prod" {
  name = "prod"
  cidr = "10.0.0.0/16"
}
```

## Resources

The provider manages the full network and compute topology:

| Resource | Purpose |
| --- | --- |
| `infra_vpc` | Virtual private cloud (bridge fabric). |
| `infra_subnet` | Subnet within a VPC. |
| `infra_security_group` | Firewall group. |
| `infra_security_group_rule` | Ingress/egress rule. |
| `infra_ip_address` | Reserved IP address. |
| `infra_igw` | Internet gateway. |
| `infra_route` | Static route. |
| `infra_kms_keyring` | KMS keyring. |
| `infra_kms_key` | KMS key (used to encrypt disks). |
| `infra_disk` | Persistent disk; `kms_key_id` encrypts it. |
| `infra_disk_file` | File injected into a disk. |
| `infra_compute` | Compute instance with attached disks. |

## Example: a VPC with an encrypted-disk compute instance

```hcl
resource "infra_vpc" "prod" {
  cidr = "10.0.0.0/16"
}

resource "infra_subnet" "pub" {
  vpc_id = infra_vpc.prod.id
  cidr   = "10.0.1.0/24"
  type   = "public"
}

resource "infra_security_group" "web" {
  vpc_id = infra_vpc.prod.id
  name   = "web"
}

resource "infra_security_group_rule" "https" {
  security_group_id = infra_security_group.web.id
  direction         = "ingress"
  protocol          = "tcp"
  port              = 443
}

resource "infra_kms_keyring" "ring" {
  name = "prod"
}

resource "infra_kms_key" "disk" {
  keyring_id = infra_kms_keyring.ring.id
  name       = "disk-encryption"
}

resource "infra_disk" "data" {
  size_mb    = 1024
  kms_key_id = infra_kms_key.disk.id
}

resource "infra_compute" "web01" {
  subnet_id         = infra_subnet.pub.id
  security_group_id = infra_security_group.web.id
  image             = "nginx:latest"
  cpu               = 1
  memory_mb         = 512
  disks = [{
    disk_id    = infra_disk.data.id
    mount_path = "/var/data"
  }]
}
```

Per-attribute reference pages are generated from the schema by `make docs-gen`
(tfplugindocs).
