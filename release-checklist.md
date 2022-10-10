# Release checklist

## Create a release

* Update [CHANGELOG.md](CHANGLOG.md)
* Update Helm chart template
  * In Chart.yaml, update `version`, `appVersion`, and
    `annotations.artifacthub.io/changes`
  * In Chart.yaml, update `artifacthub.io/changes` annotation based on changelog
* Update version compatibility matrix in [docs/index.md](docs/index.md)
* Build OperatorHub bundle
* Commit to `master`
* Branch to a release branch
* Tag a release (vX.Y.Z) on that branch
* Ensure the container becomes available on [Quay](https://quay.io/repository/backube/snapscheduler?tab=tags)

## Release updated Helm chart

* Package the Helm chart  
  `$ helm package helm/snapscheduler`
* Add it to the backube chart repo

## Release on OperatorHub

* Create bundle for operatorhub  
  `$ make bundle CHANNELS="candidate,stable" DEFAULT_CHANNEL=stable
  IMG=quay.io/backube/snapscheduler:2.0.0 VERSION=2.0.0`
  * `CHANNELS` is the list of channels that this bundle will be a part of
  * `VERSION` is the operator version (on operatorhub)
  * `DEFAULT_CHANNEL` is the channel that users will get by default
  * `IMG` is the container image + tag that will be deployed by the bundle
  * In the CSV, add the operator image: `.metadata.annotations.containerImage: quay.io/backube/snapscheduler:2.0.0`
  * In `bundle/metadata/annotations.yaml`, add the proper annotation to restrict
    which OpenShift version catalogs it will be added to:

    ```yaml
    # https://redhat-connect.gitbook.io/certified-operator-guide/ocp-deployment/operator-metadata/bundle-directory/managing-openshift-versions
    # Single version means that version and greater
    # Single version preceded by "=" means ONLY that version
    # Range is also permitted
    com.redhat.openshift.versions: "v4.7"
    ```

* Add it to the [community
  repo](https://github.com/k8s-operatorhub/community-operators/tree/main/operators/snapscheduler)
  by copying the bundle directory in as a new subdir named after the version
* Do the same for the [OpenShift
  repo](https://github.com/redhat-openshift-ecosystem/community-operators-prod/tree/main/operators/snapscheduler)
