# Container image to build
IMAGE := backube/snapscheduler
# Version of gometalinter to install (if asked)
GOMETALINTER_VERSION := v3.0.0
# Version of operator-sdk to install (if asked)
OPERATOR_SDK_VERSION := v0.5.0


.PHONY: all
all: image

.PHONY: dep-check
dep-check:
	dep check

.PHONY: dep-vendor
dep-vendor:
	dep ensure -v --vendor-only

.PHONY: generate
ZZ_GENERATED := $(shell find pkg -name 'zz_generated*')
ZZ_GEN_SOURCE := $(shell find pkg -name '*_types.go')
$(ZZ_GENERATED): $(ZZ_GEN_SOURCE)
	operator-sdk generate k8s
	operator-sdk generate openapi
generate: $(ZZ_GENERATED)

.PHONY: image
BUILDDATE := $(shell date -u '+%Y-%m-%dT%H:%M:%S.%NZ')
VERSION := $(shell git describe --tags --dirty 2> /dev/null || git describe --always --dirty)
image: generate
	operator-sdk build $(IMAGE) \
	  --docker-build-args "--build-arg builddate=$(BUILDDATE) --build-arg version=$(VERSION)"

.PHONY: install-dep
install-dep:
	curl -L https://raw.githubusercontent.com/golang/dep/master/install.sh | bash

GOBINDIR := $(shell go env GOPATH)/bin
.PHONY: install-gometalinter
install-gometalinter:
	curl -L 'https://raw.githubusercontent.com/alecthomas/gometalinter/master/scripts/install.sh' \
	  | bash -s -- -b "${GOBINDIR}" "${GOMETALINTER_VERSION}"

.PHONY: install-operator-sdk
OPERATOR_SDK_URL := https://github.com/operator-framework/operator-sdk/releases/download/$(OPERATOR_SDK_VERSION)/operator-sdk-$(OPERATOR_SDK_VERSION)-x86_64-linux-gnu
install-operator-sdk:
	curl -L "${OPERATOR_SDK_URL}" > "${GOBINDIR}/operator-sdk"
	chmod a+x "${GOBINDIR}/operator-sdk"

.PHONY: lint
lint: dep-check
	./.travis/pre-commit.sh
	gometalinter -j4 \
          --sort path --sort line --sort column \
          --deadline=24h --enable="gofmt" --vendor \
          --exclude="zz_generated.deepcopy.go" ./...
