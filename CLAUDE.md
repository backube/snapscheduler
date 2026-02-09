# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working
with code in this repository.

## Project Overview

SnapScheduler is a Kubernetes operator that provides scheduled snapshots
for CSI-based volumes. It uses a custom SnapshotSchedule CRD to define
snapshot schedules with cron-like syntax and retention policies.

The operator watches SnapshotSchedule resources and automatically creates
VolumeSnapshots for matching PVCs based on the schedule, while also
managing snapshot expiration according to retention policies.

## Commands

### Development

```bash
# Build the operator binary
make build

# Run unit tests (includes lint, generate manifests, and tests with coverage)
make test

# Run tests for a specific package
make test TEST_PACKAGES=./internal/controller

# Run linting only
make lint

# Run Helm chart linting
make helm-lint

# Generate manifests (CRDs, RBAC)
make manifests

# Generate DeepCopy methods
make generate

# Run operator locally against current kubeconfig cluster
make run

# Install CRDs into cluster
make install

# Uninstall CRDs from cluster
make uninstall
```

### Testing

```bash
# Run end-to-end tests (requires cluster with SnapScheduler running)
make test-e2e

# Setup local Kind cluster with CSI hostpath driver for E2E testing
hack/setup-kind-cluster.sh

# Build and deploy operator to Kind cluster
hack/run-in-kind.sh
```

### Container Images

```bash
# Build container image
make docker-build IMG=<image-tag>

# Push container image
make docker-push IMG=<image-tag>

# Build multi-platform image
make docker-buildx PLATFORMS=linux/amd64,linux/arm64
```

### Operator Bundle (OLM)

```bash
# Generate operator bundle
make bundle

# Build bundle image
make bundle-build

# Push bundle image
make bundle-push
```

## Architecture

### Core Components

#### Entry Point (`cmd/main.go`)

- Initializes the controller-runtime manager
- Registers the SnapshotSchedule controller
- Handles leader election and health probes
- Supports `--enable-owner-references` flag to optionally set owner
  references on VolumeSnapshots

#### API Types (`api/v1/snapshotschedule_types.go`)

- `SnapshotSchedule`: Main CRD defining snapshot schedules
- `SnapshotScheduleSpec`: Defines schedule (cron format), PVC selector,
  retention policy, and snapshot template
- `SnapshotRetentionSpec`: Defines retention via `expires` (duration) or
  `maxCount` (max snapshots per PVC)
- `SnapshotTemplateSpec`: Allows customizing VolumeSnapshot labels and
  VolumeSnapshotClass

#### Controller (`internal/controller/snapshotschedule_controller.go`)

- Reconciles SnapshotSchedule objects on every change and periodically
  (max 5 minutes)
- Finds PVCs matching `claimSelector` label selector
- Uses `robfig/cron` library to determine next scheduled snapshot time
- Creates VolumeSnapshots with labels:
  - `snapscheduler.backube/schedule`: Name of the schedule
  - `snapscheduler.backube/when`: Scheduled time in `YYYYMMDDHHMM` format
- Updates status with `lastSnapshotTime` and `nextSnapshotTime`
- Requeues reconciliation at the next scheduled snapshot time
  (capped at 5 minutes)

#### Snapshot Expiration (`internal/controller/snapshots_expire.go`)

- Called during each reconciliation to delete expired snapshots
- Supports two retention modes:
  - Time-based: Deletes snapshots older than `expires` duration
  - Count-based: Keeps only the most recent `maxCount` snapshots per PVC
- Only manages VolumeSnapshots created by SnapScheduler
  (identified by labels)

### Reconciliation Flow

1. Fetch SnapshotSchedule object
1. Parse and validate cron schedule
1. List PVCs in the schedule's namespace matching `claimSelector`
1. For each PVC:
  - Check if a snapshot should be created based on schedule and last
    snapshot time
  - Create VolumeSnapshot with appropriate labels and (optionally)
    owner reference
1. Expire old snapshots based on retention policy
1. Update status conditions and snapshot times
1. Requeue reconciliation for the next scheduled snapshot time

### Important Labels

All VolumeSnapshots created by SnapScheduler have these labels:

- `snapscheduler.backube/schedule`: Links snapshot to its schedule
- `snapscheduler.backube/when`: Scheduled time for the snapshot
  (format: `200601021504`)

These labels are critical for:

- Identifying snapshots managed by a specific schedule
- Determining snapshot age for expiration
- Avoiding duplicate snapshot creation

## Testing Framework

The project uses Ginkgo/Gomega for unit tests and KUTTL for end-to-end
tests.

### Unit Tests (`internal/controller/*_test.go`)

- Uses envtest to run tests against a real API server
- Test suite setup in `suite_test.go`
- Tests cover snapshot creation, expiration logic, and edge cases

### E2E Tests (`test-kuttl/`)

- Declarative tests using KUTTL (Kubernetes Test TooL)
- Tests run against a real cluster (typically Kind with CSI hostpath
  driver)
- Test scenarios: minimal schedules, label selectors, custom snapshot
  classes, multi-PVC handling
- **Shell scripts use POSIX syntax**: All test scripts in
  `test-kuttl/e2e/*/10-waitfor-snapshot.yaml` use POSIX-compliant shell
  syntax (single brackets `[ ]`, not bash double brackets `[[ ]]`) to
  ensure compatibility with any POSIX shell (dash, ash, bash, etc.)

## Development Notes

- All tools (controller-gen, kustomize, golangci-lint, etc.) are
  installed to `./bin` via `make` targets
- The operator uses controller-runtime v0.23.x and Kubernetes v0.35.x
- CRD manifests in `config/crd/bases/` are automatically copied to
  `helm/snapscheduler/templates/` during `make manifests`
- Version is set via git describe:
  `git describe --tags --dirty --match 'v*'`
- Pre-commit hooks run automatically on `git commit` (linting, formatting,
  etc.)

## Local E2E Testing Setup

### Prerequisites

To run e2e tests locally, you need:

- **kubectl**: Any recent version (tested with v1.31.0)
- **kind**: Version v0.31.0 (as specified in `.github/workflows/tests.yml`)
  - Earlier versions (e.g., v0.26.0) have containerd compatibility issues
  - Later versions should work but CI uses v0.31.0
- **Docker**: For building images and running kind
- **Helm**: Installed to `./bin/helm` via `make helm`

### Complete E2E Test Workflow

```bash
# 1. Build the operator
make build

# 2. Run unit tests first (faster feedback)
make test

# 3. Set up Kind cluster with CSI hostpath driver
hack/setup-kind-cluster.sh
# This creates a cluster with:
# - External snapshotter controller
# - CSI hostpath driver
# - Configured storage classes and snapshot classes

# 4. Build and load container image
make docker-build
docker tag quay.io/backube/snapscheduler:latest \
  quay.io/backube/snapscheduler:local-build
kind load docker-image quay.io/backube/snapscheduler:local-build

# 5. Deploy SnapScheduler to the cluster
bin/helm upgrade --install --create-namespace \
  -n backube-snapscheduler \
  --set image.tagOverride=local-build \
  --set metrics.disableAuth=true \
  --wait --timeout=5m \
  snapscheduler ./helm/snapscheduler

# 6. Run e2e tests
make test-e2e

# 7. Clean up
kind delete cluster
```

### Quick Test Iteration

For faster iteration when only test logic changes (no code changes):

```bash
# Tests run against existing cluster
make test-e2e
```

### Troubleshooting

**Kind image loading fails:**
If `kind load docker-image` fails with containerd errors, use manual loading:

```bash
docker save quay.io/backube/snapscheduler:local-build | \
  docker exec -i kind-control-plane ctr -n=k8s.io images import -
```

**Tests require bash:**
E2E tests now use POSIX shell syntax and work with any POSIX-compliant shell
(dash, bash, etc.). No special shell configuration needed.

**Cluster creation fails:**
Ensure you're using kind v0.31.0. Check with `kind version`.

**Pre-commit hooks fail:**
Pre-commit hooks run on `git commit`. Fix any reported issues before
committing. Common issues:

- Trailing whitespace: `fix end of files` hook
- YAML formatting: `yamllint` hook
- Large files: `check for added large files` hook

## Coding Conventions

### Shell Scripts

When writing shell scripts (including test scripts), use POSIX-compatible
syntax:

**DO:**

- Use single brackets for tests: `[ "$var" = "value" ]`
- Use `-z` for empty string checks: `[ -z "$var" ]`
- Use `-eq`, `-ne`, `-lt`, `-gt` for numeric comparisons: `[ "$n" -eq 5 ]`
- Always quote variables: `[ -n "$var" ]` not `[ -n $var ]`
- Use `=` for string equality (not `==`): `[ "$a" = "$b" ]`

**DON'T:**

- Use bash-specific double brackets: `[[ $var == value ]]`
- Use `==` for comparisons in `[ ]` tests
- Leave variables unquoted in tests

**Rationale:** POSIX syntax works in all shells (dash, ash, bash, zsh)
and is required for e2e tests to run in CI without special shell
configuration.

## License

The main codebase is licensed under GNU AGPL 3.0, but the `api/*`
directory has dual licensing (AGPL 3.0 + Apache 2.0) to allow broader
use of the CRD types.
