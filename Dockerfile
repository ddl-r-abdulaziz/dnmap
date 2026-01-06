# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install git for fetching dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies (with toolchain handling)
RUN go mod download || (go mod tidy && go mod download)

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o dnmap ./cmd/dnmap

# Runtime stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/dnmap .

# Run as non-root user
RUN adduser -D -u 1000 dnmap
USER dnmap

ENTRYPOINT ["/app/dnmap"]

