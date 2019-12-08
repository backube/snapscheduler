# Changelog

All notable changes to this project will be documented in this file. The format
is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/)

This project follows [Semantic Versioning](https://semver.org/)

## [Unreleased]

### Added

- Crontab-based schedule CR to take snapshots of CSI-based persistent volumes
- Label selectors to control which PVCs are selected for snapshotting
- Retention policies based on snapshot age or count
