# Build stage
FROM golang:1.24 AS builder

WORKDIR /app

# Pre-cache dependencies
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 go build -v -o /usr/local/bin/inventa ./cmd/inventa/

# Runtime stage
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

# Copy binary
COPY --from=builder /usr/local/bin/inventa /usr/local/bin/inventa

# Create config directory for mounted config
RUN mkdir -p /etc/inventa

# Expose the HTTP port (config default is 8081)
EXPOSE 8081

ENTRYPOINT ["/usr/local/bin/inventa", "-c", "/etc/inventa/config.yaml"]
