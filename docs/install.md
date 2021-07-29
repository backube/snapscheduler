# Installation

This page provides instructions to deploy the snapscheduler operator. The
operator is cluster-scoped, but its resources are namespaced. This means, a
single instance of the operator will permit snapshot scheduling for the entire
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
$ make install
/home/jstrunk/src/backube/snapscheduler/bin/controller-gen "crd:trivialVersions=true,preserveUnknownFields=false" rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
cp config/crd/bases/* helm/snapscheduler/crds
/home/jstrunk/src/backube/snapscheduler/bin/kustomize build config/crd | kubectl apply -f -
customresourcedefinition.apiextensions.k8s.io/snapshotschedules.snapscheduler.backube configured
```

### Install the operator

Once the CRD has been added to the cluster, the operator can be installed. The
 operator will be installed into the `snapscheduler-system` namespace.

```console
$ make deploy
/home/jstrunk/src/backube/snapscheduler/bin/controller-gen "crd:trivialVersions=true,preserveUnknownFields=false" rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
cp config/crd/bases/* helm/snapscheduler/crds
cd config/manager && /home/jstrunk/src/backube/snapscheduler/bin/kustomize edit set image controller=quay.io/backube/snapscheduler:latest
/home/jstrunk/src/backube/snapscheduler/bin/kustomize build config/default | kubectl apply -f -
namespace/snapscheduler-system created
customresourcedefinition.apiextensions.k8s.io/snapshotschedules.snapscheduler.backube created
serviceaccount/snapscheduler-controller-manager created
role.rbac.authorization.k8s.io/snapscheduler-leader-election-role created
clusterrole.rbac.authorization.k8s.io/snapscheduler-manager-role created
clusterrole.rbac.authorization.k8s.io/snapscheduler-metrics-reader created
clusterrole.rbac.authorization.k8s.io/snapscheduler-proxy-role created
rolebinding.rbac.authorization.k8s.io/snapscheduler-leader-election-rolebinding created
clusterrolebinding.rbac.authorization.k8s.io/snapscheduler-manager-rolebinding created
clusterrolebinding.rbac.authorization.k8s.io/snapscheduler-proxy-rolebinding created
configmap/snapscheduler-manager-config created
service/snapscheduler-controller-manager-metrics-service created
deployment.apps/snapscheduler-controller-manager created
```

Verify the operator starts:

```console
$ kubectl -n snapscheduler-system get deployment/snapscheduler-controller-manager
NAME                               READY   UP-TO-DATE   AVAILABLE   AGE
snapscheduler-controller-manager   1/1     1            1           4m15s
```

Once the operator is running, [continue on to usage](usage.md).
