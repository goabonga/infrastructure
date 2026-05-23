# Scheduling

Compute instances are placed onto nodes by the **scheduler**, a cluster-level
controller in `infra-controller-manager`. It runs under leader election, so a
single instance is active across the cluster.

## Nodes and pools

- A **node** registers a host with the control plane along with its schedulable
  `capacity` (CPUs, memory, max pods) and `labels`.
- A **node pool** groups nodes by a label `nodeSelector` and reports how many
  members are ready.

## Placement

Each reconcile pass the scheduler:

1. Recomputes every node's allocation from the compute already assigned to it
   (CPU, memory and pod counts), so allocation never drifts.
2. For each unscheduled compute (`status.nodeName` empty), selects the
   least-loaded node that
   - matches the compute's node pool selector (when `nodePoolId` is set), and
   - has free CPU, memory and pod capacity for the request.
3. Writes the chosen node to the compute's `status.nodeName`, and records the
   node's `status.allocated` and the pool's ready/total counts.

A compute that fits no node is left unscheduled and retried on the next pass.
Placement is written to the latest copy of the compute, so the agent's status
fields are preserved.

```
compute (nodePoolId?) ----> scheduler ----> status.nodeName = node-x
node-x.status.allocated += {cpu, memory, 1 pod}
```

## Node-scoped realization

The agent realizes compute only where it was placed. Set `GOA_NODE_ID` on
`infra-agent` to the node's id and the agent reconciles only the compute whose
`status.nodeName` matches; everything else is left to the node that owns it.

With `GOA_NODE_ID` unset the agent realizes every compute, which is the
single-host development default.
