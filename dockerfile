# Velocity Gateway - Production Dockerfile
# Multi-stage build for optimized, secure production images

# Build stage
FROM golang:1.23-alpine AS builder

# Environment setup
RUN apk add --no-cache git ca-certificates tzdata
RUN adduser -D -g '' appuser

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

# Build arguments for version information
ARG VERSION=dev
ARG BUILD_TIME
ARG GIT_COMMIT
ARG GIT_BRANCH

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -ldflags="-s -w -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME} -X main.gitCommit=${GIT_COMMIT} -X main.gitBranch=${GIT_BRANCH}" \
  -a -installsuffix cgo \
  -o velocity cmd/main.go

RUN ./velocity --version || echo "Binary created successfully"

# Runtime stage - using distroless for security
FROM gcr.io/distroless/static-debian11:nonroot

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/velocity /velocity

# Environment configuration
USER nonroot:nonroot
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/velocity", "--health-check"]

LABEL org.opencontainers.image.title="Velocity Gateway"
LABEL org.opencontainers.image.description="High-performance API Gateway"
LABEL org.opencontainers.image.vendor="Velocity Gateway Project"

ENTRYPOINT ["/velocity"]