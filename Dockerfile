FROM golang:1.26 AS builder

RUN apt-get update && apt-get install -y \
    gcc \
    libc-dev \
    libseccomp-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o sige main.go

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y \
    python3 \
    procps \
    libseccomp2 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /workspace

COPY --from=builder /app/sige /usr/local/bin/sige

COPY test /workspace/test

EXPOSE 8080

ENTRYPOINT ["sige"]
