# syntax=docker/dockerfile:1

# Dev toolchain image for local build/test/lint/run-* commands (see
# Makefile). Not used to build the production api/cron images - those
# have their own Dockerfiles (docker/api, docker/cron). The repo is
# bind-mounted over /app at runtime (see compose.yaml's backend-dev
# service), so the COPY below only exists to warm the module cache
# into an image layer.
FROM golang:1.25-alpine

# build-base (gcc/musl-dev) is required for cgo, which `go test -race`
# needs - the production api/cron images disable cgo and don't need it.
RUN apk add --no-cache git make build-base
ENV CGO_ENABLED=1

# The repo is bind-mounted at runtime and owned by the host user, which
# git (and `go build`'s VCS stamping) otherwise refuses to touch as
# root.
RUN git config --system --add safe.directory /app

WORKDIR /app

COPY src/backend/go.mod src/backend/go.sum src/backend/
RUN go -C src/backend mod download
RUN go install golang.org/x/tools/cmd/goimports@latest

CMD ["sleep", "infinity"]
