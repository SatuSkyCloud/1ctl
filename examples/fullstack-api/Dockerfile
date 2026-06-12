# ── SatuSky Cloud Build Dockerfile ───────────────────────────────────
# Builds on the SatuSky platform via Kaniko (no local Docker required).
# Cloud builds produce linux/amd64 images; multi-arch images used with
# --image are pulled as-is (kubelet picks the right variant).
# ──────────────────────────────────────────────────────────────────────

# ── Stage 1: Build ───────────────────────────────────────────────────
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src

# Cache dependencies in a separate layer.
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build a static binary.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /app ./main.go

# ── Stage 2: Runtime ─────────────────────────────────────────────────
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata curl

# Create non-root user.
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Volume mount point for persistent uploads.
RUN mkdir -p /data/uploads && chown appuser:appgroup /data/uploads

COPY --from=builder /app /app

USER appuser
EXPOSE 8080

# Health check that SatuSky smoke checks will hit.
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1

ENTRYPOINT ["/app"]
