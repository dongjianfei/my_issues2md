# ============================================================
# issue2md - Production Dockerfile
# Multi-stage build for minimal, secure container image
# ============================================================

# --- Stage 1: Builder ---
FROM golang:1.26-alpine AS builder

# Install git and ca-certificates (needed for fetching modules and HTTPS calls)
RUN apk add --no-cache git ca-certificates

WORKDIR /src

# Use China Go module proxy for reliable downloads
ENV GOPROXY=https://goproxy.cn,direct

# Dependency cache: copy go.mod/go.sum first, download deps separately
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and build
COPY . .
ARG TARGETOS=linux
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w" -trimpath -o /out/issue2md ./cmd/issue2md/

# --- Stage 2: Final (distroless-style minimal image) ---
FROM alpine:3.21 AS final

# Install ca-certificates for HTTPS API calls to GitHub
RUN apk add --no-cache ca-certificates \
    && addgroup -S appgroup \
    && adduser -S appuser -G appgroup

# Copy only the compiled binary from builder
COPY --from=builder /out/issue2md /usr/local/bin/issue2md

# Run as non-root user
USER appuser

ENTRYPOINT ["issue2md"]
