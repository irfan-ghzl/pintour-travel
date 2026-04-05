# ── Stage 1: Build Go binary ──────────────────────────────────────────────────
FROM golang:1.24-alpine AS go-builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /pintour-server ./cmd/server

# ── Stage 2: Minimal runtime image ────────────────────────────────────────────
FROM alpine:3.21

WORKDIR /app

# Non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

COPY --from=go-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=go-builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=go-builder /pintour-server /app/pintour-server

USER appuser

EXPOSE 8080

ENTRYPOINT ["/app/pintour-server"]
