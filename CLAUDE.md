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

## Development Notes

- All tools (controller-gen, kustomize, golangci-lint, etc.) are
  installed to `./bin` via `make` targets
- The operator uses controller-runtime v0.23.x and Kubernetes v0.35.x
- CRD manifests in `config/crd/bases/` are automatically copied to
  `helm/snapscheduler/templates/` during `make manifests`
- Version is set via git describe:
  `git describe --tags --dirty --match 'v*'`

## License

The main codebase is licensed under GNU AGPL 3.0, but the `api/*`
directory has dual licensing (AGPL 3.0 + Apache 2.0) to allow broader
use of the CRD types.
