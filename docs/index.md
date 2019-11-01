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

## Contents

* [Installing snapscheduler](install.md)
* [Using the scheduler](usage.md)
