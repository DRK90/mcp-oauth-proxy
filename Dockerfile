# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

RUN go mod tidy

# Copy source code
COPY . .

# Accept build arguments
ARG VERSION=dev
ARG BUILD_TIME=unknown

# Build the application with version info
RUN CGO_ENABLED=0 go build \
    -ldflags="-X main.version=${VERSION} -X main.buildTime=${BUILD_TIME} -s -w" \
    -o oauth-proxy .

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests and apply security patches
RUN apk --no-cache add ca-certificates tzdata && apk upgrade --no-cache

# Create non-root user
RUN addgroup -g 1001 -S oauth && \
    adduser -u 1001 -S oauth -G oauth

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/oauth-proxy .

# Create the SQLite data dir and hand /app to the non-root user (default DB is
# /app/data/oauth_proxy.db; mount a volume there for persistence).
RUN mkdir -p /app/data && chown -R oauth:oauth /app

# Switch to non-root user
USER oauth

# Expose port
EXPOSE 8080

# Run the application
CMD ["./oauth-proxy"]
