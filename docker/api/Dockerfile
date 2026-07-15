# syntax=docker/dockerfile:1

# Shared base: module cache, common system deps. Both the dev and build
# stages branch from here so `go mod download` is only ever done once.
FROM golang:1.25-alpine AS base
WORKDIR /app
RUN apk add --no-cache git ca-certificates tzdata
COPY go.mod go.sum ./
RUN go mod download

# ---- development ----
# No source is copied in - bind-mount the repo at /app (see compose.yaml's
# dev override or `docker run -v $(pwd):/app`) and this just `go run`s the
# API from the mounted source, so edits take effect on container restart
# without rebuilding the image. Not optimized for size; that's the
# production stage's job.
FROM base AS dev
WORKDIR /app/src/backend
EXPOSE 8091
CMD ["go", "run", "./cmd/api", "-config=config/config.json"]

# ---- build ----
# Compiles a fully static binary (CGO_ENABLED=0) so the production stage
# below can run it on scratch with no libc at all.
FROM base AS build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./src/backend/cmd/api

# ---- production ----
# scratch + only what's needed to run: the static binary, CA certs (the
# eBay/Telegram adapters make outbound HTTPS calls), tzdata, migrations,
# and config. No shell, no package manager, no alpine base at all - this
# is the smallest this image can realistically get. Runs as a non-root
# numeric UID since scratch has no /etc/passwd to resolve a username
# against.
FROM scratch AS production
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=build /out/api /api
COPY --from=build /app/src/backend/migrations /migrations
COPY --from=build /app/src/backend/config/config.json /config.json
WORKDIR /
USER 65532:65532
EXPOSE 8091
ENTRYPOINT ["/api"]
