# ── Stage 1: Builder ──────────────────────────────────────────────────────────
FROM golang:1.23-alpine AS builder

# ca-certificates penting agar binary bisa konek ke database via SSL
RUN apk add --no-cache git ca-certificates

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Compile — CGO_ENABLED=0 wajib agar binary benar-benar standalone di scratch
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags="-w -s" \
    -o taskflow-api \
    ./cmd/server/main.go

# ── Stage 2: Runtime (scratch — image ~10MB) ───────────────────────────────────
FROM scratch

# Ambil certs dari builder agar aplikasi bisa konek ke PostgreSQL/External API
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binary ke root
COPY --from=builder /build/taskflow-api /taskflow-api

EXPOSE 8080

ENTRYPOINT ["/taskflow-api"]