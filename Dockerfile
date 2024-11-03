FROM golang:1.23.2-bookworm AS builder
ARG VERSION
LABEL version=$VERSION
ENV EBPF_VERSION=$VERSION

WORKDIR /app

ENV GOSUMDB=off \
    GOINSECURE=* \
    GOPRIVATE=* \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

RUN apt-get update && apt-get install -y \
    llvm \
    clang \
    libbpf-dev \
    build-essential \
    linux-headers-generic \
    git \
    curl \
    && rm -rf /var/lib/apt/lists/*

RUN curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.1/install.sh | bash \
    && . $HOME/.nvm/nvm.sh \
    && nvm install v23.1.0 \
    && nvm use v23.1.0

RUN ln -s /usr/include/x86_64-linux-gnu/asm /usr/include/asm

COPY . .

RUN . $HOME/.nvm/nvm.sh \
    && mkdir -p /app/web \
    && cd web \
    && rm -rf node_modules package-lock.json \
    && npm install \
    && npm run build

RUN cd internal/ebpf \
    && go generate \
    && cd ../.. \
    && go build -o ebpf-firewall \
        -ldflags="-s -w" \
        -trimpath

FROM debian:bookworm-slim
WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/ebpf-firewall .

ENTRYPOINT ["/app/ebpf-firewall"]