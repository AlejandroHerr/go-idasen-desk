# Build stage
FROM golang:1.24.3-alpine AS builder

# Set build arguments
ARG APP_NAME=api
ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_TIME=unknown
ARG ENVIRONMENT=production  # Default to production for Docker builds

WORKDIR /app

# Copy the Go modules manifests
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build with version information
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-X github.com/AlejandroHerr/go-idasen-desk/version.Version=${VERSION} \
              -X github.com/AlejandroHerr/go-idasen-desk/version.Commit=${COMMIT} \
              -X github.com/AlejandroHerr/go-idasen-desk/version.BuildTime=${BUILD_TIME} \
              -X github.com/AlejandroHerr/go-idasen-desk/version.Environment=${ENVIRONMENT}" \
    -o /app/bin/${APP_NAME} ./cmd/${APP_NAME}

# Final stage
FROM alpine:3

ARG APP_NAME=api

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/bin/${APP_NAME} /app/${APP_NAME}

# Create an entrypoint script that uses the actual value of APP_NAME
RUN echo '#!/bin/sh' > /entrypoint.sh && \
    echo "exec /app/${APP_NAME} \"\$@\"" >> /entrypoint.sh && \
    chmod +x /entrypoint.sh

# Use the script as entrypoint
ENTRYPOINT ["/entrypoint.sh"]
