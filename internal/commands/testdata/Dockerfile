# Stage 1: Build the Go application
FROM golang:1.23-alpine AS builder

# Move to working directory (/build).
WORKDIR /build

# Copy and download dependency using go mod.
COPY go.mod go.sum ./
RUN go mod download

# Copy the code into the container.
COPY . .

# Set necessary environment variables needed for our image and build the API server.
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -ldflags="-s -w" -o apiserver cmd/app/main.go

# Stage 2: Production image
FROM alpine:latest

# Copy necessary dependencies from builder stage
COPY --from=builder /build/apiserver /
COPY --from=builder /build/templates /templates
COPY --from=builder /build/platform/migrations /platform/migrations

# Install additional dependencies needed
RUN apk add --no-cache \
    curl \
    openssl \
    git \
    docker \
    jq \
    && curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl \
    && chmod +x ./kubectl \
    && mv ./kubectl /usr/local/bin

# Expose necessary port.
EXPOSE 8080

# Command to run when starting the container.
ENTRYPOINT ["/apiserver"]
