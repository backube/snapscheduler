#! /bin/bash

set -e -o pipefail

# cd to top dir
scriptdir="$(dirname "$(realpath "$0")")"
cd "$scriptdir/.."

make docker-build

KIND_TAG=local-build
IMAGE="quay.io/backube/snapscheduler"

docker tag "${IMAGE}:latest" "${IMAGE}:${KIND_TAG}"
kind load docker-image "${IMAGE}:${KIND_TAG}"

kubectl create ns backube-snapscheduler
helm install -n backube-snapscheduler --set image.tagOverride=${KIND_TAG} snapscheduler ./helm/snapscheduler
