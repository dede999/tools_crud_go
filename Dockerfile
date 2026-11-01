# Stage 1: Build the static Go binary
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install certificates and git if needed
RUN apk add --no-cache ca-certificates

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy application source code
COPY . .

# Build statically linked binary with optimizations (strip debug symbols)
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-w -s" -o /bin/api ./cmd/api

# Stage 2: Minimal, secure runtime container
FROM alpine:3.20 AS runner

# Add non-root user for security best practices
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy trusted CA certificates and compiled binary
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /bin/api /app/api

EXPOSE 8080

USER appuser

ENTRYPOINT ["/app/api"]
