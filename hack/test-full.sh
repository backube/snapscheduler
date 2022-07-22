#! /bin/bash

set -e -o pipefail

SCRIPT_DIR="$(dirname "$(realpath "$0")")"

START_TIME="$(date +%s)"

# Ensure all utilities are installed/built fresh
rm -f "${SCRIPT_DIR}/../bin/*"

# Setup test cluster
"${SCRIPT_DIR}/setup-kind-cluster.sh"

CLUSTER_SETUP_DONE="$(date +%s)"

# Start operator
"${SCRIPT_DIR}/run-in-kind.sh"

OPERATOR_SETUP_DONE="$(date +%s)"

# Run all the tests
make -C "${SCRIPT_DIR}/.." test test-e2e

TESTS_DONE="$(date +%s)"

kind delete cluster

cat - <<STATS

========================================
Tests completed
----------------------------------------
   Cluster setup:      $(printf "%4ds" $((CLUSTER_SETUP_DONE - START_TIME)))
   Operator setup:     $(printf "%4ds" $((OPERATOR_SETUP_DONE - CLUSTER_SETUP_DONE)))
   Test duration:    + $(printf "%4ds" $((TESTS_DONE - OPERATOR_SETUP_DONE)))
----------------------------------------
Total elapsed time:    $(printf "%4ds" $((TESTS_DONE - START_TIME)))
========================================
STATS
