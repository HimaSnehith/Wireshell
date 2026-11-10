# syntax=docker/dockerfile:1
FROM golang:1.24-alpine AS builder

WORKDIR /src

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev

# Copy module definitions and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build static Linux binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /wireshell ./cmd/wireshell

# Final minimal runtime image
FROM alpine:3.20

RUN apk add --no-cache ca-certificates iproute2 iptables bash

WORKDIR /root
COPY --from=builder /wireshell /usr/local/bin/wireshell

# WireGuard default UDP port
EXPOSE 51820/udp

ENTRYPOINT ["/usr/local/bin/wireshell"]
CMD ["up"]
