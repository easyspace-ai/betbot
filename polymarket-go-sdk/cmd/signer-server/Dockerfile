# This Dockerfile is designed to be built from the project ROOT.
# Command: docker build -t polymarket-signer -f cmd/signer-server/Dockerfile .

# Build stage
FROM golang:1.24-alpine AS builder
WORKDIR /app

# Install git and certificates
RUN apk add --no-cache git ca-certificates

# Copy go mod and sum files from the root
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire source tree (needed because we import from /pkg)
COPY . .

# Build the binary targeting the signer package
# We use CGO_ENABLED=0 for a static binary that works in a bare alpine image
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/signer-server ./cmd/signer-server

# Run stage
FROM alpine:latest
RUN apk add --no-cache ca-certificates
WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/signer-server .

# Default configuration via env vars (can be overridden)
ENV PORT=8080
EXPOSE 8080

# Run
CMD ["./signer-server"]