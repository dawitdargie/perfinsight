# Stage 1: Build
FROM golang:latest AS builder
WORKDIR /build

# Copy dependency files first (layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the collector binary
RUN CGO_ENABLED=0 GOOS=linux go build \
	-ldflags="-s -w" \
	-o collector \
	./cmd/collector/

# Stage 2: Run
FROM alpine:latest

# Install certificates for HTTPS and wget for health checks
RUN apk --no-cache add ca-certificates wget

WORKDIR /app

# Copy only the binary from builder stage
COPY --from=builder /build/collector .

EXPOSE 9000

# Health check
HEALTHCHECK \
	--interval=5s \
	--timeout=3s \
	--start-period=10s \
	--retries=3 \
	CMD wget --quiet --tries=1 --spider http://localhost:9000/health || exit 1

CMD ["./collector"]