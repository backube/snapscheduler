# SnapScheduler

[![Build
Status](https://travis-ci.com/backube/snapscheduler.svg?branch=master)](https://travis-ci.com/backube/snapscheduler)
[![Go Report
Card](https://goreportcard.com/badge/github.com/backube/snapscheduler)](https://goreportcard.com/report/github.com/backube/snapscheduler)
[![codecov](https://codecov.io/gh/backube/snapscheduler/branch/master/graph/badge.svg)](https://codecov.io/gh/backube/snapscheduler)
[![Docker Repository on
Quay](https://quay.io/repository/backube/snapscheduler/status "Docker Repository
on Quay")](https://quay.io/repository/backube/snapscheduler)

snapscheduler provides scheduled snapshotting capabilities for Kubernetes
CSI-based volumes.

Interested in giving it a try? [Check out the
docs.](https://backube.github.io/snapscheduler/)

Have feedback? Got questions? Having trouble? [![Gitter
chat](https://badges.gitter.im/backube/snapscheduler.png)](https://gitter.im/backube/snapscheduler)

## Helpful links

- [snapscheduler Changelog](CHANGELOG.md)
- [Contributing guidelines](https://github.com/backube/.github/blob/master/CONTRIBUTING.md)
- [Organization code of conduct](https://github.com/backube/.github/blob/master/CODE_OF_CONDUCT.md)

## Licensing

This project is licensed under the [GNU AGPL 3.0 License](LICENSE) with the following
exception:

- The files within the `pkg/apis/*` directories are additionally licensed under
  Apache License 2.0. This is to permit snapscheduler's CustomResource types to
  be used by a wider range of software.
- Documentation is made available under the [Creative Commons
  Attribution-ShareAlike 4.0 International license (CC BY-SA
  4.0)](https://creativecommons.org/licenses/by-sa/4.0/)
