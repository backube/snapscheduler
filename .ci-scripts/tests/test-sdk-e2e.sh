#! /bin/bash

set -e -o pipefail

KUBECONFIG="${KUBECONFIG:-${HOME}/.kube/config}"
SCRIPT_DIR="$(dirname "$(realpath "$0")")"
TOP="$(realpath "${SCRIPT_DIR}/../..")"

# https://github.com/operator-framework/operator-sdk/blob/master/doc/test-framework/writing-e2e-tests.md#running-go-test-directly-not-recommended
# Run the e2e, but not using the operator-sdk binary. This requires ensuring flags are correctly set.
# See https://github.com/operator-framework/operator-sdk/blob/master/pkg/test/framework.go for flags.
# operator-sdk test local --go-test-flags="-v -parallel=100" "./test/e2e"
go test "${TOP}/test/e2e/..." -root="${TOP}" -kubeconfig="${KUBECONFIG}" -globalMan /dev/null -namespacedMan /dev/null -v -parallel=100
