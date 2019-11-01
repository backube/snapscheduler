# Installation

This page provides instructions to deploy the snapscheduler operator. The
operator and its resources are namespaced, so it can be deployed, configured,
and used with relatively few permissions. This also allows it to run
independently in different namespaces.

## Installation via OLM

Eventually, installation via OLM will be supported... for now, please use the
manual steps, below.

## Manual installation

Manual installation consists of several steps:

* Installing the CRD for snapshotschedules
* Installing the operator (and service account) into the application's
  namespace.

### Install the CRD

Prior to installing the operator, the CustomResourceDefinition for
snapshotschedules needs to be added to the cluster. This operation only needs to
be performed once per cluster, but it does require elevated permissions (to add
the CRD).

Install the CRD:

```
$ kubectl apply -f deploy/crds/snapscheduler_v1alpha1_snapshotschedule_crd.yaml
customresourcedefinition.apiextensions.k8s.io/snapshotschedules.snapscheduler.backube created
```

### Install the operator

Once the CRD has been installed, the operator can be added to an application's
namespace. Below, we assume the target namespace is `myns`.

Create the service account, role, and role binding for the operator:

```
$ kubectl -n myns apply -f deploy/service_account.yaml
serviceaccount/snapscheduler created
$ kubectl -n myns apply -f deploy/role.yaml
role.rbac.authorization.k8s.io/snapscheduler created
$ kubectl -n myns apply -f deploy/role_binding.yaml
rolebinding.rbac.authorization.k8s.io/snapscheduler created
```

Start the operator:

```
$ kubectl -n myns apply -f deploy/operator.yaml
deployment.apps/snapscheduler created
```

Verify the operator starts:

```
$ kubectl -n myns get deployment/snapscheduler
NAME            READY   UP-TO-DATE   AVAILABLE   AGE
snapscheduler   1/1     1            1           49s
```

Once the operator is running, [continue on to usage](usage.md).
