# SIGE — Sandbox de Isolamento para Execução de Código Não Confiável

Sandbox em Go que executa código enviado por terceiros sob isolamento de
kernel: **cgroups v2** (RAM, CPU, PIDs), **namespaces** (PID, NET, UTS, IPC,
mount), **pivot_root** com raiz somente-leitura, **seccomp BPF** em modo
allowlist e rebaixamento para um usuário sem privilégio nem capabilities.
Suporta Python, Bash, C e C++. Projeto de TCC.

> **⚠️ O SIGE só roda dentro de Docker — tanto a API quanto o CLI.**
>
> Fora de um container, ele exigiria `root` no seu host, alteraria a árvore de
> cgroups da máquina (competindo com o systemd) e montaria o `/usr`, `/bin` e
> `/etc` **do host** dentro do sandbox. O container não é empacotamento: é a
> barreira que torna aceitável executar código não confiável. Para depurar em
> modo nativo, use uma VM descartável.

---

## Começando

**Pré-requisitos:** Docker com Compose v2.21+ (`docker compose`, sem hífen) e
host Linux com **cgroup v2 unificado**. Go, `libseccomp` e os compiladores vão
todos dentro da imagem.

```bash
docker compose up --build
```

Sem passo prévio: os limites padrão vêm embutidos no binário e a chave da API é
gerada no primeiro start, aparecendo no log:

```
[SIGE]   X-API-Key: 99b1d54121271ddf...
```

Primeira chamada:

```bash
curl -X POST http://localhost:8080/execute \
  -H "X-API-Key: <a chave do log>" \
  -d '{"language":"python","code":"print(2+2)"}'
```

```json
{
  "mode": "interpreter",
  "result": "completed",
  "passed_count": 1,
  "total_count": 1,
  "execution": {
    "stdout": "4\n",
    "stderr": "",
    "duration_ms": 15,
    "memory_peak_bytes": 8388608,
    "exit_code": 0,
    "status": "success"
  }
}
```

Operação:

```bash
docker compose logs -f     # acompanhar
docker compose restart     # reiniciar (mantém a chave)
docker compose down        # parar
docker compose down -v     # parar e descartar a chave gerada
```

Para fixar sua própria chave, defina `SIGE_API_KEY` (aceita um arquivo `.env`
ao lado do `docker-compose.yml`).

---

## CLI

Executa um comando avulso no sandbox, sem passar pela API:

```bash
docker compose run --rm sige-api \
  run-task --mem 64 --timeout 5 -- python3 -c "print('Hello Sandbox')"
```

Com o servidor já no ar, use o container existente:

```bash
docker compose exec sige-api \
  /opt/sige/sige run-task --mem 64 --timeout 5 -- python3 -c "print('Hello')"
```

Flags: `--mem` (MB), `--cpu` (`"50"` ou `"50%"`), `--timeout` (s),
`--tmp-limit`, `--file-limit`, `--nofile-limit`. Veja `--help` para a lista
completa.

---

## API

### `POST /execute`

Requer o cabeçalho `X-API-Key`. O comportamento depende do modo de operação
(`SIGE_API_MODE`):

| Modo | O que faz |
| :--- | :--- |
| `interpreter` *(padrão)* | Executa e devolve o resultado bruto. Não compara saída nem aceita parâmetros de correção. |
| `single_evaluation` | Compara `stdout` com `expected_stdout` → `PASS` / `FAIL`. Aceita limites customizados. |
| `multi_evaluation` | Avalia contra uma lista de `test_cases`, até 4 sandboxes em paralelo. Retorna aprovados e a primeira falha. |

### Campos da requisição

| Campo | Tipo | Obrigatório | Descrição |
| :--- | :--- | :--- | :--- |
| `language` | `string` | **Sim** | `python`/`python3`, `bash`/`sh`, `c`, `cpp`/`c++`. |
| `code` | `string` | *Condicional* | Código-fonte. Obrigatório se `file_base64` estiver ausente. |
| `file_base64` | `string` | *Condicional* | Código-fonte em Base64. |
| `filename` | `string` | Não | Nome do arquivo (só letras, dígitos, `.`, `-`, `_`). |
| `stdin` | `string` | Não | Entrada padrão do programa. |
| `expected_stdout` | `string` | `single_evaluation` | Saída esperada. |
| `test_cases` | `array` | `multi_evaluation` | `[{ "stdin": "...", "expected_stdout": "..." }]` |
| `memory_mb`, `cpu`, `timeout_sec`, `tmp_limit_mb`, `max_file_size_mb`, `max_open_files` | — | Não | Limites por requisição; aceitos apenas nos modos de avaliação e sempre limitados pelos tetos do servidor. |

### Status da execução

O campo `execution.status` (espelhado em `error_type` quando há falha):

| Status | Significado |
| :--- | :--- |
| `success` | Terminou normalmente. |
| `timeout` | Estourou o limite de tempo; o cgroup inteiro foi morto. |
| `oom` | Estourou o limite de memória. |
| `output_limit_exceeded` | Passou de 1 MB de `stdout`/`stderr`; a execução foi encerrada. |
| `file_size_exceeded` | Tentou gravar arquivo acima do limite. |
| `compilation_error` | C/C++ não compilou; o `stderr` do compilador vai na resposta. |
| `failed` | Terminou com erro ou foi morto por sinal — inclusive por violação de seccomp. |

Erros de entrada retornam **400**; ausência de chave configurada no servidor,
**503**; chave inválida, **401**. Erros de compilação retornam **200** com
`error_type: "compilation_error"` — a requisição foi processada com sucesso.

---

## Configuração

### Autenticação

A chave é resolvida uma única vez no startup, nesta ordem:

1. Docker Secret em `/run/secrets/sige_api_key`
2. `SIGE_API_KEY`
3. Chave persistida de execução anterior (`/var/lib/sige/api_key`)
4. Uma chave nova, gerada, persistida e impressa no log

**A autenticação nunca é desabilitada por omissão** — `/execute` executa código
arbitrário, então "sem chave" não pode virar "sem autenticação". O passo 4
existe para a imagem subir sem configuração e ainda assim exigir credencial. O
`docker-compose.yml` monta um volume em `/var/lib/sige` para a chave sobreviver
ao recriar o container.

Para desligar a autenticação em desenvolvimento, o opt-in é explícito:
`SIGE_ALLOW_UNAUTHENTICATED=true`.

### Onde fica a configuração

Tudo se configura pelo `docker-compose.yml` — **não há arquivo a criar ou
montar**. A configuração efetiva é resolvida em três camadas, da menor para a
maior precedência:

1. os padrões embutidos no binário
2. um `config.json` no diretório de trabalho, **se existir** (opcional)
3. as variáveis `SIGE_*`

Cada camada só sobrepõe o que define: um `config.json` com um único campo não
zera o resto, e uma variável ausente não altera nada. Uma variável malformada
derruba o servidor **no startup**, com mensagem explícita, em vez de falhar na
primeira requisição.

Para mudar algo, edite o `docker-compose.yml` ou use um `.env`:

```bash
SIGE_MEMORY_MB=200 SIGE_TIMEOUT_SEC=12 docker compose up -d
```

### Servidor

| Variável | Padrão | Descrição |
| :--- | :--- | :--- |
| `SIGE_API_KEY` | — | Chave da API. Se vazia, uma é gerada automaticamente. |
| `SIGE_STATE_DIR` | `/var/lib/sige` | Onde a chave gerada é persistida. |
| `SIGE_ALLOW_UNAUTHENTICATED` | `false` | Roda sem chave (**apenas desenvolvimento**). |
| `SIGE_TRUSTED_PROXIES` | vazio | IPs de proxy confiáveis, separados por vírgula. Só para estes o `X-Real-IP` é respeitado no rate limit. |
| `SIGE_WORKSPACE_DIR` | `/workspace` | Base onde o workspace de cada execução é criado. |
| `TCC_EXECUTABLE` | o próprio binário | Binário reexecutado para montar o sandbox. No container, `/opt/sige/sige-launch` — a cópia com *file capabilities*. Alterar quebra o isolamento. |

### Modo e limites padrão

| Variável | Padrão | Descrição |
| :--- | :--- | :--- |
| `SIGE_API_MODE` | `interpreter` | `interpreter`, `single_evaluation` ou `multi_evaluation`. |
| `SIGE_MEMORY_MB` | `50` | RAM por execução. |
| `SIGE_CPU` | `50%` | Cota de CPU. |
| `SIGE_TIMEOUT_SEC` | `5` | Tempo máximo de execução. |
| `SIGE_TMP_LIMIT_MB` | `64` | Tamanho do `/tmp` (tmpfs) no sandbox. |
| `SIGE_MAX_FILE_SIZE_MB` | `15` | Maior arquivo que o código pode gravar. |
| `SIGE_MAX_OPEN_FILES` | `256` | Descritores abertos simultâneos. |

### Tetos do servidor

Limite máximo que uma requisição pode pedir nos modos de avaliação. Um valor
acima é **reduzido ao teto**, e ausente ou zerado nunca significa "ilimitado".

| Variável | Padrão |
| :--- | :--- |
| `SIGE_CEILING_MEMORY_MB` | `512` |
| `SIGE_CEILING_CPU_PERCENT` | `100` |
| `SIGE_CEILING_TIMEOUT_SEC` | `30` |
| `SIGE_CEILING_TMP_LIMIT_MB` | `256` |
| `SIGE_CEILING_MAX_FILE_SIZE_MB` | `64` |
| `SIGE_CEILING_MAX_OPEN_FILES` | `512` |

### Limites fixos

Não configuráveis pelo cliente nem por variável:

| Limite | Valor |
| :--- | :--- |
| Corpo da requisição | 10 MB |
| `code` / `stdin` / `expected_stdout` (cada) | 1 MB |
| `file_base64` | 2 MB |
| `test_cases` por requisição | 20 |
| Sandboxes simultâneos | 4 |
| `stdout` / `stderr` por execução | 1 MB |
| Compilação C/C++ | 256 MB RAM, 10 s, 64 MB `/tmp` |
| Processos por sandbox | 50 |

---

## Testes

```bash
go test ./...     # unitários; os que exigem privilégio se auto-pulam
```

A suíte adversarial em `test/real_cases/` (26 casos) exercita o isolamento de
verdade contra o servidor no ar — fork bomb, escape de namespace, vazamento de
arquivo na compilação, inundação de saída, processos órfãos. Veja
[`test/real_cases/README.md`](test/real_cases/README.md) para o que cada caso
verifica e como interpretar os que são mortos pelo seccomp.

---

## Manutenção

As imagens base são fixadas por **digest**, não por tag, para o build ser
reprodutível. Atualizar é, portanto, um passo deliberado:

```bash
docker pull golang:1.26 && docker image inspect golang:1.26 --format '{{index .RepoDigests 0}}'
docker pull debian:bookworm-slim && docker image inspect debian:bookworm-slim --format '{{index .RepoDigests 0}}'
# substituir os digests no Dockerfile e reconstruir
```

Antes de cada release:

```bash
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

Após qualquer mudança no mecanismo de isolamento, rode a suíte adversarial —
é o que de fato verifica se as garantias continuam valendo.

---

## Estrutura do repositório

| Caminho | Conteúdo |
| :--- | :--- |
| `cmd/` | CLI: `api`, `run-task`, `cgroups-version` e os subcomandos internos do sandbox. |
| `internal/sandbox/` | Motor de isolamento: namespaces, `pivot_root`, seccomp, rlimits. |
| `internal/api/` | Servidor HTTP, validação, workspace e os três modos de avaliação. |
| `internal/cgroups/` | Hierarquia cgroup v2 e delegação. |
| `test/real_cases/` | Suíte adversarial. |
| `docker-compose.yml` | **O único ambiente de execução suportado.** |
