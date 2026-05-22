# Architecture overview

infrastructure separates a control plane from a data plane, mirroring the way
Kubernetes splits API/controllers from kubelets.

```
Clients                       Control plane                 Data plane
-------                       -------------                 ----------
terraform-provider-infra ->   infra-api (REST)        ->    infra-agent (per host)
infra (cli)              ->        |                              |
                                   v                              v
                              state store                   Linux kernel
                              (file | etcd | postgres)      (netns, bridges, VXLAN,
                                   ^                          iptables, cgroups v2,
infra-idp (identity)               |                          dm-crypt, dnsmasq, OCI)
infra-exporter (metrics) <- controller-manager
                              (scheduler + reconcilers)
```

## Control plane

- **infra-api** records the desired `spec` and observed `status` of every
  resource and serves them over REST.
- **infra-controller-manager** schedules resources onto hosts and runs the
  cluster-level reconcile loops.

## Data plane

- **infra-agent** runs on each host, watches the resources assigned to it and
  reconciles them against the kernel. See [realization](realization.md) for how a
  declared topology becomes bridges, namespaces, cgroups and encrypted disks.

## State

The store is pluggable: a file backend for single-host development, etcd for
multi-host clusters, and Postgres for identity data.

See the [resource model](resource-model.md) for the spec/status contract.
