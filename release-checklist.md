# Release checklist

## Create a release

* Update Helm chart template
  * In Chart.yaml, update `version`, `appVersion`, and
    `annotations.artifacthub.io/changes`
* Create bundle for operatorhub  
  `$ make bundle CHANNELS="candidate,stable" DEFAULT_CHANNEL=stable
  IMG=quay.io/backube/snapscheduler:v2.0.0 VERSION=2.0.0`
  * `CHANNELS` is the list of channels that this bundle will be a part of
  * `VERSION` is the operator version (on operatorhub)
  * `DEFAULT_CHANNEL` is the channel that users will get by default
  * `IMG` is the container image + tag that will be deployed by the bundle
* Update version compatibility matrix in [docs/index.md](docs/index.md)
* Commit to `master`
* Branch to a release branch
* Tag a release (vX.Y.Z) on that branch
* Ensure the container becomes available on [Quay](https://quay.io/repository/backube/snapscheduler?tab=tags)

## Release updated Helm chart

* Package the Helm chart  
  `$ helm package helm/snapscheduler`
* Add it to the backube chart repo

## Release on OperatorHub

* Add it to the [community
  repo](https://github.com/k8s-operatorhub/community-operators/tree/main/operators/snapscheduler)
  by copying the bundle directory in as a new subdir named after the version
* Do the same for the [OpenShift
  repo](https://github.com/redhat-openshift-ecosystem/community-operators-prod/tree/main/operators/snapscheduler)
