# Post-MVP Polish — Go backend multi-stage Docker image
# Stage 1: Build
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Download dependencies first (layer cache)
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build static binary — CGO_ENABLED=0 (modernc.org/sqlite is pure-Go)
RUN CGO_ENABLED=0 GOOS=linux \
    go build -ldflags="-w -s" -o fenix ./cmd/fenix

# Stage 2: Runtime — minimal Alpine image
FROM alpine:3.19

RUN apk add --no-cache ca-certificates curl

WORKDIR /app

COPY --from=builder /app/fenix ./fenix

# Data directory for SQLite
RUN mkdir -p /data

EXPOSE 8080

HEALTHCHECK --interval=15s --timeout=5s --start-period=20s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1

CMD ["./fenix", "serve", "--port", "8080"]
