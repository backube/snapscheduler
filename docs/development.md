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
  - Can be installed by: `make install-operator-sdk`
- [golangci-lint](https://github.com/golangci/golangci-lint)
  - Check
    [Makefile](https://github.com/backube/snapscheduler/blob/master/Makefile)
    for the proper version to use
  - Can be installed by: `make install-golangci`
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

Since the operator is based on the operator-sdk, the SDK's commands are used for
much of the build/run process. A Makefile is provided to wrap those commands and
ensure proper flags are provided.

To build the operator's binary and container, use the `image` target:

```console
$ make image
operator-sdk build quay.io/backube/snapscheduler \
  --go-build-args "-ldflags -X=github.com/backube/snapscheduler/version.Version=291d1fd-dirty" \
  --image-build-args "--build-arg builddate=2019-11-10T02:56:55.314848329Z \
  --build-arg version=291d1fd-dirty"
INFO[0030] Building OCI image quay.io/backube/snapscheduler
...
Successfully built 688dcc82bd71
Successfully tagged quay.io/backube/snapscheduler:latest
INFO[0041] Operator build complete.
```

The above can then be pushed to a custom repository and tested in-cluster.

However, during development, it's much quicker to run the operator locally. For
that, just use the SDK from the top-level directory:

```console
$ operator-sdk up local --kubeconfig=/path/to/kubeconfig --namespace ""
...
```

## Modifying the code

When changing the code, if the `types.go` file is modified, the deepcopy and
openapi files need to be regenerated. Use `make generate` for that prior to
running the operator.

After making modifications, and before committing code, please ensure both the
tests and linters pass or the PR will be rejected by the CI system:

```console
$ make test
...
$ make lint
...
```

## CI system & E2E testing

The CI system (GitHub Actions) checks each PR by running both the linters + unit
tests (mentioned above) and end-to-end tests. These tests are run across a
number of kubernetes versions (see `KUBERNETES_VERSIONS` in
[`.github/workflows/tests.yml`](https://github.com/backube/snapscheduler/blob/master/.github/workflows/tests.yml)).

The e2e tests can be found in the
[`tests/e2e`](https://github.com/backube/snapscheduler/blob/master/tests/e2e)
directory and are based on the operator-sdk's e2e testing library.

### Running E2E locally just like CI

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
$ ./.ci-scripts/tests/test-sdk-e2e.sh
=== RUN   TestSnapscheduler
=== RUN   TestSnapscheduler/Minimal_schedule
=== PAUSE TestSnapscheduler/Minimal_schedule
=== RUN   TestSnapscheduler/Snapshot_labeling
=== PAUSE TestSnapscheduler/Snapshot_labeling
=== RUN   TestSnapscheduler/Custom_snapclass
...
PASS
ok    github.com/backube/snapscheduler/test/e2e 70.640s
```
