# Container image to build
IMAGE := quay.io/backube/snapscheduler
# Version of golangci-lint to install (if asked)
GOLANGCI_VERSION := v1.17.1
# Version of operator-sdk to install (if asked)
OPERATOR_SDK_VERSION := v0.10.0
GOBINDIR := $(shell go env GOPATH)/bin


.PHONY: all
all: image

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
	  --go-build-args "-ldflags -X=github.com/backube/SnapScheduler/version.Version=$(VERSION)" \
	  --image-build-args "--build-arg builddate=$(BUILDDATE) --build-arg version=$(VERSION)"

.PHONY: install-golangci
GOLANGCI_URL := https://install.goreleaser.com/github.com/golangci/golangci-lint.sh
install-golangci:
	curl -fL ${GOLANGCI_URL} | sh -s -- -b ${GOBINDIR} ${GOLANGCI_VERSION}

.PHONY: install-operator-sdk
OPERATOR_SDK_URL := https://github.com/operator-framework/operator-sdk/releases/download/$(OPERATOR_SDK_VERSION)/operator-sdk-$(OPERATOR_SDK_VERSION)-x86_64-linux-gnu
install-operator-sdk:
	curl -fL "${OPERATOR_SDK_URL}" > "${GOBINDIR}/operator-sdk"
	chmod a+x "${GOBINDIR}/operator-sdk"

.PHONY: lint
lint: generate
	./.travis/pre-commit.sh
	golangci-lint run --no-config --deadline 30m --disable-all -v \
	  --enable=deadcode \
	  --enable=errcheck \
	  --enable=gocyclo \
	  --enable=goimports \
	  --enable=gosec \
	  --enable=gosimple \
	  --enable=govet \
	  --enable=ineffassign \
	  --enable=misspell \
	  --enable=staticcheck \
	  --enable=structcheck \
	  --enable=typecheck \
	  --enable=unconvert \
	  --enable=unused \
	  --enable=varcheck \
	  ./...

.PHONY: test
test: coverage.txt

.PHONY: codecov
codecov: coverage.txt
	curl -fL https://codecov.io/bash | bash -s

coverage.txt: generate
	go test -covermode=atomic -coverprofile=coverage.txt  ./...
