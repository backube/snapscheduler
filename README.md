# SnapScheduler

[![Build
Status](https://github.com/backube/snapscheduler/workflows/Tests/badge.svg)](https://github.com/backube/snapscheduler/actions?query=branch%3Amaster+workflow%3ATests+)
[![Go Report
Card](https://goreportcard.com/badge/github.com/backube/snapscheduler)](https://goreportcard.com/report/github.com/backube/snapscheduler)
[![codecov](https://codecov.io/gh/backube/snapscheduler/branch/master/graph/badge.svg)](https://codecov.io/gh/backube/snapscheduler)

SnapScheduler provides scheduled snapshots for Kubernetes CSI-based volumes.

## Quickstart

Install:

```console
$ helm repo add backube https://backube.github.io/helm-charts/
"backube" has been added to your repositories

$ kubectl create namespace backube-snapscheduler
namespace/backube-snapscheduler created

$ helm install -n backube-snapscheduler snapscheduler backube/snapscheduler
NAME: snapscheduler
LAST DEPLOYED: Mon Jul  6 15:16:41 2020
NAMESPACE: backube-snapscheduler
STATUS: deployed
...
```

Keep 6 hourly snapshots of all PVCs in `mynamespace`:

```console
$ kubectl -n mynamespace apply -f - <<EOF
apiVersion: snapscheduler.backube/v1
kind: SnapshotSchedule
metadata:
  name: hourly
spec:
  retention:
    maxCount: 6
  schedule: "0 * * * *"
EOF

snapshotschedule.snapscheduler.backube/hourly created
```

In this example, there is 1 PVC in the namespace, named `data`:

```console
$ kubectl -n mynamespace get pvc
NAME   STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS      AGE
data   Bound    pvc-c2e044ab-1b24-496a-9569-85f009892ccf   1Gi        RWO            csi-hostpath-sc   9s
```

At the top of each hour, a snapshot of that volume will be automatically
created:

```console
$ kubectl -n mynamespace get volumesnapshots
NAME                       AGE
data-hourly-202007061600   82m
data-hourly-202007061700   22m
```

## More information

Interested in giving it a try? [Check out the
docs.](https://backube.github.io/snapscheduler/)

The operator can be installed from:

- [Artifact
  Hub](https://artifacthub.io/package/chart/backube-helm-charts/snapscheduler)
- [Helm Hub](https://hub.helm.sh/charts/backube/snapscheduler)
- [OperatorHub.io](https://operatorhub.io/operator/snapscheduler)

Other helpful links:

- [SnapScheduler Changelog](CHANGELOG.md)
- [Contributing guidelines](https://github.com/backube/.github/blob/master/CONTRIBUTING.md)
- [Organization code of conduct](https://github.com/backube/.github/blob/master/CODE_OF_CONDUCT.md)

## Licensing

This project is licensed under the [GNU AGPL 3.0 License](LICENSE) with the following
exceptions:

- The files within the `api/*` directories are additionally licensed under
  Apache License 2.0. This is to permit SnapScheduler's CustomResource types to
  be used by a wider range of software.
- Documentation is made available under the [Creative Commons
  Attribution-ShareAlike 4.0 International license (CC BY-SA
  4.0)](https://creativecommons.org/licenses/by-sa/4.0/)
