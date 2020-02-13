# snapscheduler

The snapscheduler operator enables taking scheduled snapshots of Kubernetes
persistent volumes.

## Introduction

The snapscheduler operator takes snapshots of CSI-based PersistentVolumes
according to a configurable Cron-like schedule. These schedules also permit
configuring the retention of the automated snapshots. The goal is to allow
simple automated snapshotting policies like, "Retain 7 daily snapshots of the
PVCs matching *(some selector)*."

Please see the [full documentation](https://backube.github.io/snapscheduler/)
for more information.

## Requirements

- Kubernetes >= 1.13
- CSI-based storage driver that supports snapshots (i.e. has the
  `CREATE_DELETE_SNAPSHOT` capability)

## Usage

The snapscheduler operator is a "cluster-level" operator. A single instance will
watch `snapshotschedules` across all namespaces in the cluster. **Running more
than one instance of the scheduler at a time is not supported.**

### Installation

It is recommended to install the operator into the `backube-snapscheduler`
namespace, though any namespace may be used.

```console
$ helm install --namespace backube-snapscheduler snapscheduler backube/snapscheduler
NAME: snapscheduler
LAST DEPLOYED: Mon Nov 25 17:38:26 2019
NAMESPACE: backube-snapscheduler
STATUS: deployed
REVISION: 1
TEST SUITE: None
NOTES:
Thank you for installing snapscheduler!

The snapscheduler operator is now installed in the backube-snapscheduler
namespace, and snapshotschedules should be enabled cluster-wide.

See https://backube.github.io/snapscheduler/usage.html to get started.

Schedules can be viewed via:
$ kubectl -n <mynampspace> get snapshotschedules
...
```

### Examples

The schedule for snapshotting is controlled by the
`snapshotschedules.snapscheduler.backube` Custom Resource. This is a namespaced
resource that applies only to the PersistentVolumeClaims in its namespace. Below
is a simple example. See the [usage
documentation](https://backube.github.io/snapscheduler/usage.html) for full
details.

Keep 7 daily snapshots of all PVCs in a given namespace:

```yaml
---
apiVersion: snapscheduler.backube/v1
kind: SnapshotSchedule
metadata:
  name: daily
spec:
  retention:
    maxCount: 7
  schedule: "@daily"
  snapshotTemplate:
    snapshotClassName: csi-ebs
```

## Configuration

The following optional parameters in the chart can be configured, either by
using `--set` on the command line or via a `values.yaml` file. In the general
case, the defaults should be sufficient.

- `replicaCount`: `2`
  - The number of replicas of the operator to run. Only one is active at a time
    via leader election.
- `image.repository`: `quay.io/backube/snapscheduler`
  - The location of the operator container image
- `image.tagOverride`: `""`
  - If set, it will override the operator container image tag. The default tag
    is set per chart version and can be viewed (as `appVersion`) via `helm show
    chart`.
- `image.pullPolicy`: `IfNotPresent`
  - Overrides the container image pull policy
- `imagePullSecrets`: none
  - May be set if pull secret(s) are needed to retrieve the operator image
- `serviceAccount.create`: `true`
  - Whether to create the ServiceAccount for the operator
- `serviceAccount.name`: none
  - Override the name of the operator's ServiceAccount
- `podSecurityContext`: none
  - Allows setting the security context for the operator pod
- `securityContext`: none
  - Allows setting the operator container's security context
- `resources`: requests for 10m CPU and 100Mi memory; no limits
  - Allows overriding the resource requests/limits for the operator pod
- `nodeSelector`: none
  - Allows applying a node selector to the operator pod
- `tolerations`: none
  - Allows applying tolerations to the operator pod
- `affinity`: none
  - Allows setting the operator pod's affinity
