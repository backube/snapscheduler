# Installation

This page provides instructions to deploy the snapscheduler operator. The
operator is cluster-scoped, but its resources are namespaced. This means, a
single instance of teh operator will permit snapshot scheduling for the entire
cluster. However, the snapshot schedules are unique per-namespace.

## Installation via Helm v3

There is a [Helm](https://helm.sh) chart available to install the snapscheduler
operator. For production deployments, this is the recommended method of
deployment.

First, add the backube chart repository to your list of repos in Helm:

```console
$ helm repo add backube https://backube.github.io/helm-charts/
"backube" has been added to your repositories
```

Then, install the operator's chart:

```console
$ kubectl create namespace backube-snapscheduler
namespace/backube-snapscheduler created

$ helm install -n backube-snapscheduler snapscheduler backube/snapscheduler
NAME: snapscheduler
LAST DEPLOYED: Tue Dec 10 09:32:26 2019
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
```

## Manual installation

Manual installation consists of several steps:

* Installing the CRD for snapshotschedules
* Installing the operator.

### Install the CRD

Prior to installing the operator, the CustomResourceDefinition for
snapshotschedules needs to be added to the cluster. This operation only needs to
be performed once per cluster, but it does require elevated permissions (to add
the CRD).

Install the CRD:

```console
$ kubectl apply -f deploy/crds/snapscheduler.backube_snapshotschedules_crd.yaml
customresourcedefinition.apiextensions.k8s.io/snapshotschedules.snapscheduler.backube created
```

### Install the operator

Once the CRD has been added to the cluster, the operator can be installed. The
RBAC configuration for the operator, as defined in the `deploy/` directory,
assumes the operator will be installed into the `backube-snapscheduler`
namespace.

First, create the namespace:

```console
$ kubectl create ns backube-snapscheduler
namespace/backube-snapscheduler created
```

Create the service account, role, and role binding for the operator:

```console
$ kubectl apply -f deploy/service_account.yaml
serviceaccount/snapscheduler created
$ kubectl apply -f deploy/role.yaml
clusterrole.rbac.authorization.k8s.io/snapscheduler created
$ kubectl apply -f deploy/role_binding.yaml
clusterrolebinding.rbac.authorization.k8s.io/snapscheduler created
```

Start the operator:

```console
$ kubectl apply -f deploy/operator.yaml
deployment.apps/snapscheduler created
```

Verify the operator starts:

```console
$ kubectl -n backube-snapscheduler get deployment/snapscheduler
NAME            READY   UP-TO-DATE   AVAILABLE   AGE
snapscheduler   2/2     2            2           49s
```

Once the operator is running, [continue on to usage](usage.md).
