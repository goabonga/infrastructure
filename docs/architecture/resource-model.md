# Resource model

Every resource is a record with three parts, following the Kubernetes
convention:

- **metadata** - identity and lifecycle: `uid`, `name`, `organizationId`,
  `projectId`, `labels`, `annotations`, `ownerRefs`, `finalizers`,
  `generation`, `deletionTimestamp`, `createdAt`.
- **spec** - the desired state declared by the client.
- **status** - the observed state recorded by the controllers and agent:
  `phase`, `conditions` and `observedGeneration`.

## Phases

```
Pending -> Reconciling -> Ready
                       -> Error
           Deleting    -> Terminated
```

## Conditions

Conditions report orthogonal facts about a resource, for example `Ready`,
`Synced`, `Healthy`, `Progressing`, `Degraded`, `Scheduled` and `Bound`. Each
carries a status, reason and timestamp.

## Ownership and cleanup

`ownerRefs` express parent/child relationships so deleting a parent cascades to
its children. `finalizers` block deletion until a controller has released the
underlying kernel resources.

## Resource types

VPC, subnet, internet gateway, route, peering, security group (+ rule), IP
address, compute, disk, disk file, DNS zone, DNS record, KMS keyring, KMS key,
secret (+ version), SSL CA, SSL cert, WAF policy (+ rule), ACL policy (+ rule),
load balancer (+ backend).
