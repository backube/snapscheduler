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

Recommended:

- markdownlint
- yamllint
- shellcheck

## Building the code

Since the operator is based on the operator-sdk, the SDK's commands are used for
much of the build/run process. A Makefile is provided to wrap those commands and
ensure proper flags are provided.

To build the operator's binary and container, use the `image` target:

```
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

```
operator-sdk up local --kubeconfig=/path/to/kubeconfig --namespace <ns-to-watch>
```

## Modifying the code

When changing the code, if the `types.go` file is modified, the deepcopy and
openapi files need to be regenerated. Use `make generate` for that prior to
running the operator.

After making modifications, and before committing code, please ensure both the
tests and linters pass or the PR will be rejected by the CI system:

```
make test
make lint
```
