#! /bin/bash

set -e -o pipefail

make docker-build

kind load docker-image "quay.io/backube/snapscheduler"

kubectl create ns backube-snapscheduler
helm install -n backube-snapscheduler --set image.tagOverride=latest snapscheduler ./helm/snapscheduler
