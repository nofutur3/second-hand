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
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o api ./cmd/api

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/api .

# Copy migrations
COPY --from=builder /app/migrations ./src/backend/migrations

# Copy config file
COPY --from=builder /app/config.json ./src/backend/config/config.json

# Expose API port
EXPOSE 8091

# Run the API
CMD ["./api"]
