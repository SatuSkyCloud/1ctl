FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o server .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE 8080
# SatuSky runs workloads with runAsNonRoot, so an image that defaults to root
# is rejected by the kubelet with CreateContainerConfigError before it starts.
USER 65532:65532
CMD ["./server"]
