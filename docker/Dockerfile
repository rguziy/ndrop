FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy dependency files first for better layer caching.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o ndropd \
    ./cmd/ndropd

# -------------------------------------------------------
# Final image: scratch (no OS, no shell, minimal attack surface)
# -------------------------------------------------------
FROM scratch

COPY --from=builder /app/ndropd /ndropd

EXPOSE 8080

ENTRYPOINT ["/ndropd", "start"]
