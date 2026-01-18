# git-herd Docker Image - Multi-stage build
# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build arguments for version info
ARG BUILD_VERSION=dev
ARG BUILD_DATE
ARG GIT_COMMIT

# Build the binary
RUN CGO_ENABLED=0 go build -v -trimpath \
    -ldflags="-s -w -X main.version=${BUILD_VERSION} -X main.date=${BUILD_DATE} -X main.commit=${GIT_COMMIT}" \
    -o git-herd \
    ./cmd/git-herd

# Runtime stage
FROM alpine:3.23

# Install runtime dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Create non-root user
RUN adduser -D -s /bin/sh git-herd

# Copy the binary from builder
COPY --from=builder /build/git-herd /usr/local/bin/git-herd

# Ensure binary is executable
RUN chmod +x /usr/local/bin/git-herd

# Switch to non-root user
USER git-herd

# Set working directory
WORKDIR /workspace

# Add labels for better container management
LABEL org.opencontainers.image.title="git-herd"
LABEL org.opencontainers.image.description="A concurrent Git repository management tool"
LABEL org.opencontainers.image.source="https://github.com/entro314-labs/git-herd"
LABEL org.opencontainers.image.url="https://github.com/entro314-labs/git-herd"
LABEL org.opencontainers.image.vendor="entro314-labs"

# Default command
ENTRYPOINT ["/usr/local/bin/git-herd"]
CMD ["--help"]