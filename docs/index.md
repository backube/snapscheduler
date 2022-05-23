# Documentation

snapscheduler enables taking [snapshots of Kubernetes Persistent
Volumes](https://kubernetes.io/docs/concepts/storage/volume-snapshots/)
according to a defined schedule. Each schedule defines how often a new snapshot
should be created as well as the retention policy for the snapshots (i.e., the
maximum number to retain and/or for how long).

## Requirements

The snapscheduler operator is designed to work with Kubernetes [CSI-based
storage
drivers](https://kubernetes.io/blog/2019/01/15/container-storage-interface-ga/)
that are capable of taking volume snapshots.

Kubernetes version compatibility:

| snapscheduler | Kubernetes    | `snapshot.storage.k8s.io` |
|---------------|---------------|---------------------------|
| 1.0           | 1.13 -- 1.16  | `v1alpha1`                |
| 1.1           | 1.13 -- 1.21  | `v1alpha1`, `v1beta1`     |
| 1.2           | 1.13 -- 1.21  | `v1alpha1`, `v1beta1`     |
| 2.0           | 1.17 -- 1.23  | `v1beta1`                 |
| 2.1           | 1.17 -- 1.23  | `v1beta1`                 |
| 3.0           | 1.20 -- 1.24+ | `v1`                      |
| master        | 1.20 -- 1.24+ | `v1`                      |

## Contents

User documentation

- [Installing snapscheduler](install.md)
- [Using the scheduler](usage.md)
  - [Approaches to labeling PVCs](labeling.md)

Developer documentation

- [Building & running snapscheduler locally](development.md)
- [Editing the documentation](docs.md)
- [Upgrading the operator-sdk version](sdk-upgrade.md)
- [Project tracking & roadmap](roadmap.md)
