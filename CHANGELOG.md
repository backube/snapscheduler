# Changelog

All notable changes to this project will be documented in this file. The format
is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/)

This project follows [Semantic Versioning](https://semver.org/)

## [Unreleased]

## [3.2.0] - 2022-10-10

### Added

- Ability to specify a container hash instead of just a tag when deploying via
  Helm chart

### Changed

- :warning: Helm chart now directly manages CRD installation/upgrade  
  If upgrading from 3.1.0 or earlier, manual steps are required:

  ```console
  Error: UPGRADE FAILED: rendered manifests contain a resource that already
  exists. Unable to continue with update: CustomResourceDefinition
  "snapshotschedules.snapscheduler.backube" in namespace "" exists and cannot be
  imported into the current release: invalid ownership metadata; label
  validation error: missing key "app.kubernetes.io/managed-by": must be set to
  "Helm"; annotation validation error: missing key "meta.helm.sh/release-name":
  must be set to "snapscheduler"; annotation validation error: missing key
  "meta.helm.sh/release-namespace": must be set to "backube-snapscheduler"
  ```

  The above error can be fixed by adding the required labels and annotations as
  mentioned in the error message:

  ```console
  $ kubectl label crd/snapshotschedules.snapscheduler.backube app.kubernetes.io/managed-by=Helm
  customresourcedefinition.apiextensions.k8s.io/snapshotschedules.snapscheduler.backube labeled

  $ kubectl annotate crd/snapshotschedules.snapscheduler.backube meta.helm.sh/release-name=snapscheduler
  customresourcedefinition.apiextensions.k8s.io/snapshotschedules.snapscheduler.backube annotated

  $ kubectl annotate crd/snapshotschedules.snapscheduler.backube meta.helm.sh/release-namespace=backube-snapscheduler
  customresourcedefinition.apiextensions.k8s.io/snapshotschedules.snapscheduler.backube annotated
  ```

- Made CRD validation of cronspec more permissive
- Upgrade quay.io/brancz/kube-rbac-proxy to v0.13.1
- Upgrade operator-sdk to 1.23.0

## [3.1.0] - 2022-08-01

### Added

- Ability to control TopologySpreadConstraints via Helm chart

### Changed

- Upgrade quay.io/brancz/kube-rbac-proxy to v0.13.0
- Upgrade operator-sdk to 1.22.0

## [3.0.0] - 2022-04-01

### Changed

- Snapshot objects are now accessed via `snapshot.storage.k8s.io/v1` API version
- Upgrade operator-sdk to 1.18

### Removed

- Removed support for Kubernetes versions < 1.20

## [2.1.0] - 2021-12-17

### Added

- Ability to configure resource requests for RBAC proxy container when deploying
  via Helm chart.
- Ability to configure container image used for kube-rbac-proxy

### Changed

- Build w/ Go 1.17
- Upgrade kube-rbac-proxy image to 0.11.0
- Upgrade operator-sdk to 1.15

## [2.0.0] - 2021-08-03

### Changed

- Updated project scaffolding to operator-sdk 1.10
- Moved CRD to `apiextensions.k8s.io/v1`
- Added default host anti-affinity for the operator replicas
- Updated Helm Chart manifests to more closely match OSDK scaffolding

### Removed

- Removed support for Kubernetes versions < 1.17
- Removed support for `snapshot.storage.k8s.io/v1alpha1` snapshot version
- Removed node selector labels targeting `beta.kubernetes.io/arch` and
  `beta.kubernetes.io/os`

## [1.2.0] - 2021-04-05

### Changed

- Switched the operator base container to distroless

### Fixed

- Metrics weren't accessible from the snapsheduler-metrics Service

## [1.1.1] - 2020-04-24

### Fixed

- Fix crash when snapshotTemplate is not defined in schedule

## [1.1.0] - 2020-02-13

### Added

- Support Kubernetes 1.17 and `snapshot.storage.k8s.io/v1beta1` snapshot version

## [1.0.0] - 2019-12-09

### Added

- Crontab-based schedule CR to take snapshots of CSI-based persistent volumes
- Label selectors to control which PVCs are selected for snapshotting
- Retention policies based on snapshot age or count

[Unreleased]: https://github.com/backube/snapscheduler/compare/v3.2.0...HEAD
[3.2.0]: https://github.com/backube/snapscheduler/compare/v3.1.0...v3.2.0
[3.1.0]: https://github.com/backube/snapscheduler/compare/v3.0.0...v3.1.0
[3.0.0]: https://github.com/backube/snapscheduler/compare/v2.1.0...v3.0.0
[2.1.0]: https://github.com/backube/snapscheduler/compare/v2.0.0...v2.1.0
[2.0.0]: https://github.com/backube/snapscheduler/compare/v1.2.0...v2.0.0
[1.2.0]: https://github.com/backube/snapscheduler/compare/v1.1.1...v1.2.0
[1.1.1]: https://github.com/backube/snapscheduler/compare/v1.1.0...v1.1.1
[1.1.0]: https://github.com/backube/snapscheduler/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/backube/snapscheduler/releases/tag/v1.0.0
