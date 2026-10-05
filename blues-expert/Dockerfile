# Build stage (runs on the build platform and cross-compiles for the target)
FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS builder

# Install git (needed for Go's VCS stamping)
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Version injected via -ldflags at build time (e.g. a release tag). Left empty
# for ad-hoc builds so the binary falls back to the embedded VCS revision.
ARG VERSION=
ARG TARGETOS
ARG TARGETARCH

# Build the application
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build \
    -ldflags="-X main.version=${VERSION}" -o main ./blues-expert/

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates wget

# Create non-root user
RUN adduser -D -s /bin/sh appuser

WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Change ownership to non-root user
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose ports 8080 (MCP server)
EXPOSE 8080

# Run the binary
CMD ["./main", "-log-level", "debug"]
