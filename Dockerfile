# Build stage
ARG GO_VERSION=1.24.13
FROM golang:${GO_VERSION}-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o github-code-agent ./cmd/agent

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/github-code-agent .

# Copy config file template if it exists
COPY --from=builder /app/.github/code-agent.yml .github/code-agent.yml

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser && \
    chown -R appuser:appuser /app

USER appuser

# Default port (can be overridden)
EXPOSE 8001

# Run the application
CMD ["./github-code-agent"]
