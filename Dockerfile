# Estágio de build com dependências para CGO e libseccomp
FROM golang:1.26@sha256:5822931cf78fe98a97edcf73a0c54c29fa2386b99c8136468e274ae9fab8cfba AS builder

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

# Estágio final de execução
FROM debian:bookworm-slim@sha256:abd67ffcfa541b485a3dff59865ab629aa048a6c613e639d36e7456b0b229241

# Instalação de interpretadores, compiladores e bibliotecas de suporte
RUN apt-get update && apt-get install -y \
    python3 \
    gcc \
    g++ \
    libc-dev \
    procps \
    libseccomp2 \
    libcap2-bin \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /workspace

# Binário principal do daemon
COPY --from=builder /app/sige /opt/sige/sige

# Binário auxiliar com capabilities para montagem e isolamento do sandbox
RUN cp /opt/sige/sige /opt/sige/sige-launch && \
    setcap cap_sys_admin,cap_setuid,cap_setgid,cap_setpcap+ep /opt/sige/sige-launch

ENV TCC_EXECUTABLE=/opt/sige/sige-launch

COPY test /workspace/test

# Cria usuário não-privilegiado sige (UID/GID 1001)
RUN groupadd --gid 1001 sige && \
    useradd --uid 1001 --gid sige --no-create-home --shell /bin/false sige && \
    chown sige:sige /workspace

# Diretório de persistência para chave de API gerada automaticamente
RUN mkdir -p /var/lib/sige && chown sige:sige /var/lib/sige && chmod 0700 /var/lib/sige
VOLUME ["/var/lib/sige"]

EXPOSE 8080

# Healthcheck HTTP via socket TCP
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD bash -c 'exec 3<>/dev/tcp/127.0.0.1/8080 && printf "GET / HTTP/1.0\r\n\r\n" >&3 && head -c 15 <&3 | grep -q "200 OK"'

ENTRYPOINT ["/opt/sige/sige"]
CMD ["api", "-p", "8080"]
