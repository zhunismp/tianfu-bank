# ---------- Builder Stage ----------
FROM --platform=linux/arm64 golang:1.24-alpine AS builder

RUN apk add --no-cache \
    gcc \
    g++ \
    musl-dev \
    librdkafka-dev \
    pkgconfig \
    git \
    bash \
    curl

RUN go install github.com/bazelbuild/bazelisk@latest

WORKDIR /build

COPY . .

RUN bazelisk build //services/account-service:account-service

# ---------- Runtime Stage ----------
FROM alpine:3.19

# Copy certs
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy Bazel built binary
COPY --from=builder \
    /build/bazel-bin/services/account-service/account-service_/account-service \
    /account-service

EXPOSE 8080

ENTRYPOINT ["/account-service"]