# syntax=docker/dockerfile:1

FROM golang:1.24-alpine AS builder

WORKDIR /src

# Install certs for module downloads if needed.
RUN apk add --no-cache ca-certificates git

# Cache dependencies first.
COPY go.mod go.sum ./
RUN go mod download

# Copy source.
COPY . .

# Build static-ish binary.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/engine ./cmd/engine

FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

# Binary
COPY --from=builder /out/engine /app/engine

# Rules used by the engine at runtime
COPY rules /app/rules

# Default runtime envs
ENV PORT=8080
ENV RULES_DIR=./rules
ENV PERSISTENCE_BACKEND=memory

# The engine is currently a batch-style process, not an HTTP server.
ENTRYPOINT ["/app/engine"]
