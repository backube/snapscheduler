# Changelog

All notable changes to this project will be documented in this file. The format
is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/)

This project follows [Semantic Versioning](https://semver.org/)

## [Unreleased]

TBD

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

[unreleased]: https://github.com/backube/snapscheduler/compare/v1.2.0...HEAD
[1.2.0]: https://github.com/backube/snapscheduler/compare/v1.1.1...v1.2.0
[1.1.1]: https://github.com/backube/snapscheduler/compare/v1.1.0...v1.1.1
[1.1.0]: https://github.com/backube/snapscheduler/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/backube/snapscheduler/releases/tag/v1.0.0
