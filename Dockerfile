# Build Stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install certificates and build tools
RUN apk add --no-cache ca-certificates git

# Cache dependencies
COPY go.mod go.sum* ./
RUN go mod download || true

# Copy source code
COPY . .

# Build statically linked binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o /app/bin/short-go ./cmd/api

# Final Minimal Runtime Stage
FROM alpine:3.20

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata

# Create non-root user for security
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

COPY --from=builder /app/bin/short-go /app/short-go
COPY --from=builder /app/migrations /app/migrations

USER appuser

EXPOSE 8080

ENTRYPOINT ["/app/short-go"]
