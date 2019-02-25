#! /bin/bash

# Usage: push_container.sh <repo> <verbatim|version> <tag>
set -e -o pipefail

image="$1"

if [[ "x$2" == "xversion" ]]; then
        [[ "$3" =~ ^v([0-9]+.*) ]] || exit 1;
        tag="${BASH_REMATCH[1]}"
else
        tag="$3"
fi

if [[ "x${QUAY_USERNAME}" != "x" && "x${QUAY_PASSWORD}" != "x" ]]; then
        echo "$QUAY_PASSWORD" | docker login -u "$QUAY_USERNAME" --password-stdin quay.io
        finalimage="quay.io/$image:$tag"
        docker tag "$image" "$finalimage"
        docker push "$finalimage"
fi

if [[ "x${DOCKER_USERNAME}" != "x" && "x${DOCKER_PASSWORD}" != "x" ]]; then
        echo "$DOCKER_PASSWORD" | docker login -u "$DOCKER_USERNAME" --password-stdin docker.io
        finalimage="docker.io/$image:$tag"
        docker tag "$image" "$finalimage"
        docker push "$finalimage"
fi
