# Build the manager binary
FROM golang:1.18 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/

# Build
ARG version="(unknown)"
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager -ldflags -X=main.snapschedulerVersion=${version} main.go

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
