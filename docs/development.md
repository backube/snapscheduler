# Development

This page provides an overview of how to get started enhancing snapscheduler.

## Prerequisites

Required:

- A working Go environment
- Docker
- [operator-sdk](https://github.com/operator-framework/operator-sdk)
  - Check
    [Makefile](https://github.com/backube/snapscheduler/blob/master/Makefile)
    for the proper version to use
  - Can be installed by: `make operator-sdk`
- [kind](https://kind.sigs.k8s.io/)
  - Recommended for running E2E tests in combination with the CSI hostpath
    driver
  - Check
    [Makefile](https://github.com/backube/snapscheduler/blob/master/Makefile)
    for the proper version to use

Recommended:

- markdownlint
- yamllint
- shellcheck

## Building the code

It is possible to run the operator locally against a running cluster. This
enables quick turnaround during development. With a running cluster (and
properly configured kubeconfig):

Install the CRDs:

```console
$ make install
/home/jstrunk/src/backube/snapscheduler/bin/controller-gen "crd:trivialVersions=true,preserveUnknownFields=false" rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
cp config/crd/bases/* helm/snapscheduler/crds
/home/jstrunk/src/backube/snapscheduler/bin/kustomize build config/crd | kubectl apply -f -
customresourcedefinition.apiextensions.k8s.io/snapshotschedules.snapscheduler.backube created
```

Run the operator locally:

```console
$ make run
/home/jstrunk/src/backube/snapscheduler/bin/controller-gen "crd:trivialVersions=true,preserveUnknownFields=false" rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
cp config/crd/bases/* helm/snapscheduler/crds
/home/jstrunk/src/backube/snapscheduler/bin/controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./..."
/home/jstrunk/src/backube/snapscheduler/bin/golangci-lint run ./...
go run -ldflags -X=main.snapschedulerVersion=v1.1.0-105-g53576a0-dirty ./main.go
2021-07-20T13:18:58.059-0400    INFO    setup    Operator Version: v1.1.0-105-g53576a0-dirty
2021-07-20T13:18:58.059-0400    INFO    setup    Go Version: go1.16.4
2021-07-20T13:18:58.059-0400    INFO    setup    Go OS/Arch: linux/amd64
2021-07-20T13:18:58.969-0400    INFO    controller-runtime.metrics    metrics server is starting to listen    {"addr": ":8080"}
2021-07-20T13:18:58.992-0400    INFO    setup    starting manager
2021-07-20T13:18:58.993-0400    INFO    controller-runtime.manager    starting metrics server    {"path": "/metrics"}
2021-07-20T13:18:58.993-0400    INFO    controller-runtime.manager.controller.snapshotschedule    Starting EventSource    {"reconciler group": "snapscheduler.backube", "reconciler kind": "SnapshotSchedule", "source": "kind source: /, Kind="}
2021-07-20T13:18:59.094-0400    INFO    controller-runtime.manager.controller.snapshotschedule    Starting Controller    {"reconciler group": "snapscheduler.backube", "reconciler kind": "SnapshotSchedule"}
2021-07-20T13:18:59.094-0400    INFO    controller-runtime.manager.controller.snapshotschedule    Starting workers    {"reconciler group": "snapscheduler.backube", "reconciler kind": "SnapshotSchedule", "worker count": 1}
...
```

## CI system & E2E testing

The CI system (GitHub Actions) checks each PR by running both the linters + unit
tests (mentioned above) and end-to-end tests. These tests are run across a
number of kubernetes versions (see `KUBERNETES_VERSIONS` in
[`.github/workflows/tests.yml`](https://github.com/backube/snapscheduler/blob/master/.github/workflows/tests.yml)).

### Running E2E locally

The same scripts that are used in CI can be used to test and develop locally:

- The
  [`hack/setup-kind-cluster.sh`](https://github.com/backube/snapscheduler/blob/master/hack/setup-kind-cluster.sh)
  script will create a Kubernetes cluster and install the CSI hostpath driver.
  The `KUBE_VERSION` environment variable can be used to change the Kubernetes
  version. Note that this must be a specific version `X.Y.Z` that has a Kind
  container.
- The
  [`hack/run-in-kind.sh`](https://github.com/backube/snapscheduler/blob/master/hack/run-in-kind.sh)
  script will build the operator image, inject it into the Kind cluster, and use
  the local helm chart to start it.

After running the above two scripts, you should have a running cluster with a
suitable CSI driver and the snapscheduler running, ready for testing.

The E2E tests can then be executed via:

```console
$ make test-e2e
cd test-kuttl && /home/jstrunk/src/backube/snapscheduler/bin/kuttl test
=== RUN   kuttl
    harness.go:457: starting setup
    harness.go:248: running tests using configured kubeconfig.
    harness.go:285: Successful connection to cluster at: https://127.0.0.1:37729
    harness.go:353: running tests
    harness.go:74: going to run test suite with timeout of 30 seconds for each step
    harness.go:365: testsuite: ./e2e has 6 tests
=== RUN   kuttl/harness
=== RUN   kuttl/harness/custom-snapclass
=== PAUSE kuttl/harness/custom-snapclass
...
=== CONT  kuttl
    harness.go:399: run tests finished
    harness.go:508: cleaning up
    harness.go:563: removing temp folder: ""
--- PASS: kuttl (80.81s)
    --- PASS: kuttl/harness (0.00s)
        --- PASS: kuttl/harness/minimal-schedule (15.31s)
        --- PASS: kuttl/harness/label-selector-equality (78.02s)
        --- PASS: kuttl/harness/template-labels (78.02s)
        --- PASS: kuttl/harness/custom-snapclass (78.02s)
        --- PASS: kuttl/harness/multi-pvc (78.04s)
        --- PASS: kuttl/harness/label-selector-set (78.04s)
PASS
```

### Testing w/ OLM

To test the deployment of SnapScheduler w/ OLM (i.e., using the bundle that will
be consumed in OpenShift), it's necessary to build the bundle, bundle image, and
a catalog image.

Build and push the bundle and catalog images:

- IMG: The operator image that will be referenced in the bundle
- IMAGE_TAG_BASE: The base name for the bundle & catalog images (i.e.,
  foo-bundle, foo-catalog)
- CHANNELS: The list of channels that the bundle will belong to
- DEFAULT_CHANNEL: The default channel when someone installs
- VERSION: The bundle version number (likely the same as the operator version)

```console
$ make bundle bundle-build bundle-push catalog-build catalog-push IMAGE_TAG_BASE=quay.io/johnstrunk/snapscheduler CHANNELS="candidate,stable" DEFAULT_CHANNEL=stable IMG=quay.io/backube/snapscheduler:latest VERSION=2.0.0
...
```

Create a kind cluster & start OLM on it:

```console
$ hack/setup-kind-cluster.sh
...
$ bin/operator-sdk olm install
...
```

Add the new catalog image to the cluster:

```console
$ kubectl -n olm apply -f - <<EOF
---
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: snapscheduler-catalog
spec:
  sourceType: grpc
  # This should match the image and version from above
  image: quay.io/johnstrunk/snapscheduler-catalog:v2.0.0
EOF
```

Create a subscription for the operator so that it will install:

```console
$ kubectl -n operators apply -f - <<EOF
---
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: snapscheduler
spec:
  name: snapscheduler
  sourceNamespace: olm
  # Channel needs to match the channel in the bundle
  channel: stable
  # Needs to match the CatalogSource
  source: snapscheduler-catalog
  installPlanApproval: Automatic
```
