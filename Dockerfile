# Frontend build stage
FROM node:22-alpine AS frontend
WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Go build stage
FROM golang:1.24 AS builder
WORKDIR /app

# Pre-cache dependencies
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source and frontend build output
COPY . .
COPY --from=frontend /app/cmd/inventa/web-dist ./cmd/inventa/web-dist
RUN CGO_ENABLED=0 go build -v -o /usr/local/bin/inventa ./cmd/inventa/

# Create the config directory (cannot use RUN in distroless — ship as file)
RUN mkdir -p /etc/inventa && touch /etc/inventa/.config-dir

# Runtime stage
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app

# Copy binary
COPY --from=builder /usr/local/bin/inventa /usr/local/bin/inventa

# Copy pre-created config directory
COPY --from=builder --chown=nonroot:nonroot /etc/inventa /etc/inventa

# Expose the HTTP port (config default is 8081)
EXPOSE 8081

ENTRYPOINT ["/usr/local/bin/inventa", "-c", "/etc/inventa/config.yaml"]
