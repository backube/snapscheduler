# Container image to build
IMAGE := quay.io/backube/snapscheduler
# Version of golangci-lint to install (if asked)
GOLANGCI_VERSION := v1.23.3
# Version of operator-sdk to install (if asked)
OPERATOR_SDK_VERSION := v0.15.1
GOBINDIR := $(shell go env GOPATH)/bin


.PHONY: all
all: image

.PHONY: docs
docs:
	cd docs && bundle update
	cd docs && PAGES_REPO_NWO=backube/snapscheduler bundle exec jekyll serve -l

.PHONY: generate
ZZ_GENERATED := $(shell find pkg -name 'zz_generated*')
ZZ_GEN_SOURCE := $(shell find pkg -name '*_types.go')
$(ZZ_GENERATED): $(ZZ_GEN_SOURCE)
	operator-sdk generate crds
	operator-sdk generate k8s
	openapi-gen --logtostderr=true -o "" -i ./pkg/apis/snapscheduler/v1 -O zz_generated.openapi -p ./pkg/apis/snapscheduler/v1 -h ./hack/openapi-gen-boilerplate.go.txt -r "-"
generate: $(ZZ_GENERATED)

.PHONY: image
BUILDDATE := $(shell date -u '+%Y-%m-%dT%H:%M:%S.%NZ')
VERSION := $(shell git describe --tags --dirty --match 'v*' 2> /dev/null || git describe --always --dirty)
image: generate
	operator-sdk build $(IMAGE) \
	  --go-build-args "-ldflags -X=github.com/backube/snapscheduler/version.Version=$(VERSION)" \
	  --image-build-args "--build-arg builddate=$(BUILDDATE) --build-arg version=$(VERSION)"

.PHONY: install-golangci
GOLANGCI_URL := https://install.goreleaser.com/github.com/golangci/golangci-lint.sh
install-golangci:
	curl -fL ${GOLANGCI_URL} | sh -s -- -b ${GOBINDIR} ${GOLANGCI_VERSION}

.PHONY: install-helm
install-helm:
	curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash

.PHONY: install-openapi-gen
openapi-gen:
	go get k8s.io/kube-openapi/cmd/openapi-gen

.PHONY: install-operator-sdk
OPERATOR_SDK_URL := https://github.com/operator-framework/operator-sdk/releases/download/$(OPERATOR_SDK_VERSION)/operator-sdk-$(OPERATOR_SDK_VERSION)-x86_64-linux-gnu
install-operator-sdk:
	mkdir -p ${GOBINDIR}
	curl -fL "${OPERATOR_SDK_URL}" > "${GOBINDIR}/operator-sdk"
	chmod a+x "${GOBINDIR}/operator-sdk"

.PHONY: lint
lint: generate
	helm lint helm/snapscheduler
	golangci-lint run ./...

.PHONY: test
test: coverage.txt

.PHONY: codecov
codecov: coverage.txt
	curl -fL https://codecov.io/bash | bash -s

coverage.txt: generate
	go test -covermode=atomic -coverprofile=coverage.txt  $(shell go list ./... | grep -v /test/e2e)
