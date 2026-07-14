# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the API
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o api ./src/backend/cmd/api

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/api .

# Copy migrations (path is relative to CWD at runtime, see database.Migrate("migrations"))
COPY --from=builder /app/src/backend/migrations ./migrations

# Copy config file (path is relative to CWD at runtime, see config.Load("config.json"))
COPY --from=builder /app/src/backend/config/config.json ./config.json

# Expose API port
EXPOSE 8091

# Run the API
CMD ["./api"]
