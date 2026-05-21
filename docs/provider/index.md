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

## Resource reference

Per-resource pages are generated from the provider schema by `make docs-gen`
and published under this section. They are added as the provider implementation
lands (see the project plan).
