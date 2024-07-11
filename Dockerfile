# Build the manager binary
FROM golang:1.22@sha256:829eff99a4b2abffe68f6a3847337bf6455d69d17e49ec1a97dac78834754bd6 AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY cmd/main.go cmd/main.go
COPY api/ api/
COPY internal/controller/ internal/controller/

# Build
ARG version="(unknown)"
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o manager -ldflags -X=main.snapschedulerVersion=${version} cmd/main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]

ARG builddate="(unknown)"
ARG description="Operator to manage scheduled PV snapshots"

LABEL build-date="${builddate}"
LABEL description="${description}"
LABEL io.k8s.description="${description}"
LABEL io.k8s.displayname="snapscheduler: A snapshot scheduler"
LABEL name="snapscheduler"
LABEL summary="${description}"
LABEL vcs-type="git"
LABEL vcs-url="https://github.com/backube/snapscheduler"
LABEL vendor="Backube"
LABEL version="${version}"
