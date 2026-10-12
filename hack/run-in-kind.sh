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

# Keep the test image local so the suite doesn't block on pulling it at runtime.
docker pull busybox
busybox_platform="$(docker image inspect busybox --format '{{.Os}}/{{.Architecture}}{{if .Variant}}/{{.Variant}}{{end}}')"
busybox_archive="$(mktemp)"
trap 'rm -f "$busybox_archive"' EXIT
docker_save_help="$(docker image save --help 2>&1)"
platform_save_succeeded=false
if [[ "$docker_save_help" == *"--platform"* ]] && \
    docker image save --platform "$busybox_platform" --output "$busybox_archive" busybox; then
  platform_save_succeeded=true
fi
if [[ "$platform_save_succeeded" != true ]]; then
  docker_store_info="$(docker info --format '{{json .DriverStatus}}')"
  if [[ "$docker_store_info" == *"containerd"* ]]; then
    echo "Could not create a single-platform archive for Docker's containerd image store; ensure docker image save --platform is supported by both the client and engine." >&2
    exit 1
  fi
  # The classic image store has the platform selected by docker pull.
  docker image save --output "$busybox_archive" busybox
fi
# Use a single-platform archive to work around Kind's containerd image-store import issue:
# https://kind.sigs.k8s.io/docs/user/known-issues/#unable-to-kind-load-docker-images
kind load image-archive "$busybox_archive"

helm upgrade --install --create-namespace -n backube-snapscheduler \
    --debug \
    --set image.tagOverride=${KIND_TAG} \
    --set metrics.disableAuth=true \
    --wait --timeout=5m \
    snapscheduler ./helm/snapscheduler
