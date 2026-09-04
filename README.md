# 🛡️ SIGE — Sandbox de Isolamento para Execução Segura de Código

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26+-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go Version" />
  <img src="https://img.shields.io/badge/Linux_Kernel-cgroups_v2_|_seccomp_BPF-FCC624?style=for-the-badge&logo=linux&logoColor=black" alt="Linux Kernel" />
  <img src="https://img.shields.io/badge/Docker_Compose-v2.21+-2496ED?style=for-the-badge&logo=docker&logoColor=white" alt="Docker Compose" />
  <img src="https://img.shields.io/badge/License-Apache_2.0-blue.svg?style=for-the-badge" alt="License" />
  <img src="https://img.shields.io/badge/Security-Hardened_Sandbox-red?style=for-the-badge&logo=security" alt="Security Hardened" />
  <img src="https://img.shields.io/badge/Status-Production_Ready-brightgreen?style=for-the-badge" alt="Status" />
</p>

<p align="center">
  <strong>Motor de execução e avaliação segura de código não confiável (Online Judge / Code Runner Engine) de alto desempenho desenvolvido em Go.</strong><br />
  Isolamento profundo baseado nas primitivas do kernel Linux: <code>cgroups v2</code>, <code>namespaces</code>, <code>pivot_root</code>, <code>seccomp BPF</code> e modelo de privilégios mínimos.
</p>

---

## 📌 Sumário

- [Sobre o Projeto](#-sobre-o-projeto)
  - [O Problema](#o-problema)
  - [A Abordagem SIGE (Defesa em Profundidade)](#a-abordagem-sige-defesa-em-profundidade)
  - [Arquitetura de Isolamento](#arquitetura-de-isolamento)
- [Funcionalidades Principais](#-funcionalidades-principais)
- [Tecnologias Utilizadas](#-tecnologias-utilizadas)
- [Pré-requisitos](#-pré-requisitos)
- [Instalação e Execução](#-instalação-e-execução)
  - [1. Inicializando com Docker Compose](#1-inicializando-com-docker-compose)
  - [2. Autenticação e Primeira Execução](#2-autenticação-e-primeira-execução)
  - [3. Execução via Linha de Comando (CLI)](#3-execução-via-linha-de-comando-cli)
- [API REST](#-api-rest)
  - [Modos de Avaliação](#modos-de-avaliação)
  - [Estrutura da Requisição](#estrutura-da-requisição)
  - [Status de Execução](#status-de-execução)
- [Variáveis de Ambiente](#-variáveis-de-ambiente)
- [Estrutura de Pastas](#-estrutura-de-pastas)
- [Testes e Segurança](#-testes-e-segurança)
- [Como Contribuir](#-como-contribuir)
- [Licença e Autor](#-licença-e-autor)

---

## 📖 Sobre o Projeto

O **SIGE** é um sandbox autônomo e motor de julgamento de código projetado para executar código arbitrário enviado por terceiros com contenção rigorosa e mitigação ativa contra abusos e ataques cibernéticos.

### O Problema

Executar código arbitrário e não confiável (seja em juízes online como Codeforces/LeetCode, maratonas de programação, corretores educacionais de tarefas ou esteiras de CI/CD) expõe o host a graves vulnerabilidades:
- **Esgotamento de Recursos:** *Fork bombs*, alocações massivas de memória (*OOM*) e laços infinitos consumindo 100% da CPU.
- **Quebra de Confidencialidade e Integridade:** Acesso a arquivos confidenciais do sistema operacional hospedeiro (`/etc/passwd`, variáveis de ambiente, binários).
- **Ataques de Rede:** Varreduras internas, ataques de negação de serviço (DDoS) ou *Server-Side Request Forgery* (SSRF).
- **Fugas de Privilégios (*Privilege Escalation*):** Exploração de chamadas de sistema vulneráveis para obter acesso `root` no host.

### A Abordagem SIGE (Defesa em Profundidade)

Em vez de depender unicamente de um container Docker padrão (que compartilha privilégios e recursos do kernel), o SIGE atua como uma barreira multicamada (*Defense in Depth*):

1. **Camada Externa (Container Docker):** O SIGE opera exclusivamente dentro de um container dedicado com restrições globais de CPU, memória e descritores.
2. **Camada de Mediação (Daemon SIGE em Go):** Valida esquemas, aplica limites estritos de taxa (*rate limiting*), gerencia fila de tarefas e workspaces efêmeros.
3. **Camada Interna (Sandbox no Kernel):** Cada execução instancia um ambiente hermético criado sob demanda:
   - **cgroups v2:** Controle estrito de CPU (*throttling*), limite inegociável de RAM (com `swap` desabilitado) e teto de processos (`pids.max`).
   - **Namespaces Linux:** Isolamento completo de Processos (`PID`), Rede (`NET` sem loopback ou rotas externas), Hostname (`UTS`), IPC (`IPC`) e Montagem (`MOUNT`).
   - **Filesystem Somente-Leitura (`pivot_root`):** O código enxerga uma raiz somente-leitura com `/tmp` isolado em `tmpfs` efêmero com cota de gravação.
   - **Filtro Seccomp BPF:** Modo allowlist restritivo que bloqueia syscalls perigosas (`unshare`, `clone` desautorizado, sockets de rede, `ptrace`). Violações acionam terminação imediata do processo (`ActKillProcess`).
   - **Privilégios Mínimos:** Descarte total de capabilities do Linux e rebaixamento para o usuário não-privilegiado `nobody` (UID/GID `65534`).

### Arquitetura de Isolamento

```text
               REQUISIÇÃO (HTTP API / CLI)
                           │
                           ▼
             ┌───────────────────────────┐
             │    SIGE Daemon (Go)       │
             │  • Validação e Rate Limit │
             │  • Workspace Efêmero      │
             │  • Pool de Concorrência   │
             └─────────────┬─────────────┘
                           │ Fork & Exec com File Capabilities
                           ▼
             ┌───────────────────────────┐
             │       SIGE-LAUNCH         │
             │                           │
             │   cgroups v2 Controller   │ ──► [memory.max, cpu.max, pids.max]
             │   Linux Namespaces Setup  │ ──► [NEWPID, NEWNET, NEWNS, NEWUTS, NEWIPC]
             │   Rootfs & pivot_root     │ ──► [Read-Only Root + tmpfs isolado]
             │   rlimits enforcement     │ ──► [RLIMIT_FSIZE, RLIMIT_NOFILE]
             │   Seccomp BPF Allowlist   │ ──► [Bloqueio de syscalls perigosas]
             │   Drop Privileges         │ ──► [SetUID/SetGID nobody (65534)]
             └─────────────┬─────────────┘
                           │ execve
                           ▼
             ┌───────────────────────────┐
             │   CÓDIGO NÃO CONFIÁVEL    │
             │   Python / C / C++ / Bash │
             └───────────────────────────┘
```

---

## ⚡ Funcionalidades Principais

### 🛡️ 1. Mecanismo de Isolamento & Segurança
- **Hierarquia cgroups v2:** Aplicação de cotas de CPU (`cpu.max`), teto de memória (`memory.max`) sem fallback em swap e proteção anti-*fork bomb* (`pids.max = 50`).
- **Isolamento de Rede Total:** Namespace de rede desacoplado (`CLONE_NEWNET`) sem rotas externas ou conectividade, neutralizando SSRF e conexões reversas.
- **Rootfs Imutável & Efêmero:** Montagem de filesystem isolado com `pivot_root`, tornando `/usr`, `/bin` e `/lib` estritamente somente-leitura.
- **Controle de Disco & I/O:** Workspace isolado em `tmpfs` temporário com limitação de escrita por arquivo (`RLIMIT_FSIZE`) e descritores abertos (`RLIMIT_NOFILE`).
- **Filtro Seccomp BPF Granular:** Allowlist restritiva baseada em `libseccomp`; chamadas não autorizadas finalizam o processo no ato via `SIGSYS` / `ActKillProcess`.

### 💻 2. Suporte Multilinguagem
- **Interpretadas:** Python 3 (`python`, `python3`), Bash (`bash`, `sh`).
- **Compiladas:** C (`c`), C++ (`cpp`, `c++`) com GCC/G++.
- **Compilação Segura em Sandbox:** Processo de compilação encapsulado em limites rígidos independentes (256 MB RAM, 10 s de CPU, 64 MB em `/tmp`).

### 🚀 3. API REST & Modos de Avaliação
- **Modo `interpreter`:** Execução direta com retorno bruto de saída (`stdout`, `stderr`), tempo de CPU, pico de memória consumida e código de saída.
- **Modo `single_evaluation`:** Comparação de saída com `expected_stdout`, emitindo veredito de aprovação (`PASS`/`FAIL`).
- **Modo `multi_evaluation`:** Avaliação concorrente de até 20 casos de teste com execução paralela em até 4 sandboxes simultâneos.
- **Proteções de API:** Rate limiting dinâmico por IP via Token Bucket, validação de cabeçalhos de segurança (`nosniff`, `DENY`), sanitização de caminhos e limite de payload (10 MB).

### ⌨️ 4. Interface CLI Nativa
- Subcomando `run-task` para execução direta de rotinas no sandbox via terminal, ideal para scripts, pipelines de integração e inspeção local.

---

## 🛠️ Tecnologias Utilizadas

| Categoria | Tecnologia | Finalidade |
| :--- | :--- | :--- |
| **Linguagem & Backend** | ![Go](https://img.shields.io/badge/Go_1.26-00ADD8?style=flat-square&logo=go&logoColor=white) | Core do daemon, orquestração de processos e servidor HTTP |
| **CLI Framework** | ![Cobra](https://img.shields.io/badge/spf13%2Fcobra-v1.10-blue?style=flat-square) | Interface de linha de comando (`run-task`, `api`) |
| **Kernel & Isolamento** | ![Linux](https://img.shields.io/badge/Linux_Kernel-cgroups_v2-FCC624?style=flat-square&logo=linux&logoColor=black) | Gerenciamento de recursos via `containerd/cgroups/v3` |
| **Filtro de Syscalls** | ![Seccomp](https://img.shields.io/badge/libseccomp-BPF_Allowlist-red?style=flat-square) | Restrição de syscalls via `libseccomp-golang` |
| **Containerização** | ![Docker](https://img.shields.io/badge/Docker-24.0+-2496ED?style=flat-square&logo=docker&logoColor=white) | Contenção externa, compilação multi-stage e empacotamento |
| **Orquestração** | ![Compose](https://img.shields.io/badge/Docker_Compose-v2.21+-2496ED?style=flat-square&logo=docker&logoColor=white) | Configuração de ambiente e isolamento com `cap_add: SYS_ADMIN` |

---

## 📋 Pré-requisitos

> **⚠️ AVISO CRÍTICO DE AMBIENTE:** O SIGE **só deve ser executado via Docker**.  
> Executá-lo diretamente no host exigiria privilégios de `root`, alteraria a árvore de cgroups nativa (competindo com o `systemd`) e montaria o sistema de arquivos do seu host dentro do sandbox. O container Docker fornece a casca externa indispensável de segurança.

- **Sistema Operacional Hospedeiro:** Linux com **cgroups v2 unificado** (padrão no Ubuntu 22.04+, Debian 11+, Fedora 34+, Arch Linux).
  - Verifique executando no terminal:
    ```bash
    stat -fc %T /sys/fs/cgroup/
    # Deve retornar: cgroup2fs
    ```
  - Ou diretamente pelo binário do SIGE:
    ```bash
    ./sige cgroups-version
    # Ou via Docker:
    docker compose run --rm sige-api cgroups-version
    # Saída esperada: cgroup version: v2 (Unified)
    ```
- **Docker Engine:** Versão 24.0 ou superior.
- **Docker Compose:** Versão v2.21 ou superior (`docker compose`, sem hífen).
- *Nota:* Compiladores (gcc, g++), interpretador Python e dependências do Go já vêm empacotados na imagem Docker.

---

## 🚀 Instalação e Execução

### 1. Inicializando com Docker Compose

Clone o repositório e inicie o container:

```bash
# 1. Clonar o repositório
git clone https://github.com/Geraldoaf/sige.git
cd sige

# 2. Configurar o ambiente (opcional, defaults seguros inclusos)
cp .env.example .env

# 3. Construir a imagem e iniciar em segundo plano
docker compose up -d --build
```

### 2. Autenticação e Primeira Execução

No primeiro início, caso nenhuma chave seja fornecida, o SIGE gera uma API Key criptograficamente segura de 32 bytes e a exibe nos logs:

```bash
docker compose logs sige-api | grep "X-API-Key"
```

Saída esperada:
```text
[SIGE]   X-API-Key: 99b1d54121271ddf4c...
```

Copie a chave e faça sua primeira requisição HTTP:

```bash
curl -X POST http://127.0.0.1:8080/execute \
  -H "X-API-Key: SUA_CHAVE_AQUI" \
  -H "Content-Type: application/json" \
  -d '{
    "language": "python",
    "code": "print(sum(range(1, 101)))"
  }'
```

Resposta:
```json
{
  "mode": "interpreter",
  "result": "completed",
  "passed_count": 1,
  "total_count": 1,
  "execution": {
    "stdout": "5050\n",
    "stderr": "",
    "duration_ms": 14,
    "memory_peak_bytes": 8388608,
    "exit_code": 0,
    "status": "success"
  }
}
```

### 3. Execução via Linha de Comando (CLI)

O SIGE possui o comando `run-task` para execuções avulsas diretamente pelo terminal:

```bash
# Executando um comando isolado em um container novo
docker compose run --rm sige-api \
  run-task --mem 64 --timeout 5 -- python3 -c "print('Olá do Sandbox!')"

# Ou executando dentro do container da API já ativo
docker compose exec sige-api \
  /opt/sige/sige run-task --mem 64 --timeout 5 -- python3 -c "import os; print('PID isolado:', os.getpid())"
```

---

## 🔌 API REST

### `POST /execute`

Executa o código fornecido sob os parâmetros solicitados.

#### Modos de Avaliação (`SIGE_API_MODE`)

| Modo | Descrição | Limites Customizados por Requisição? | Avaliação Automática? |
| :--- | :--- | :---: | :---: |
| `interpreter` *(padrão)* | Execução pura e direta. Retorna saída bruta (`stdout`/`stderr`), tempo e uso de memória. | ❌ *(usa defaults do servidor)* | ❌ |
| `single_evaluation` | Avalia contra `expected_stdout`. Emite veredito `PASS` ou `FAIL`. | ✔️ | ✔️ (caso único) |
| `multi_evaluation` | Avalia bateria de `test_cases`, paralelizando até 4 sandboxes simultâneos. | ✔️ | ✔️ (múltiplos casos) |

---

#### Exemplos de Uso por Modo

##### 1. Modo `interpreter` (Execução Direta)

> 💡 **Nota:** No modo `interpreter`, o código é executado sob os limites padrão configurados no servidor (`SIGE_MEMORY_MB`, `SIGE_TIMEOUT_SEC`, etc.), não aceitando parâmetros de correção (`expected_stdout`, `test_cases`) nem sobrescrita de limites na requisição.

**Requisição (`curl`):**
```bash
curl -X POST http://127.0.0.1:8080/execute \
  -H "X-API-Key: SUA_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "language": "python",
    "code": "import sys\nlinhas = sys.stdin.read().split()\nprint(sum(map(int, linhas)))",
    "stdin": "10 20 30 40\n"
  }'
```

**Resposta:**
```json
{
  "mode": "interpreter",
  "result": "completed",
  "passed_count": 1,
  "total_count": 1,
  "execution": {
    "stdout": "100\n",
    "stderr": "",
    "duration_ms": 18,
    "memory_peak_bytes": 8388608,
    "exit_code": 0,
    "status": "success",
    "limit_memory_bytes": 52428800,
    "limit_cpu": "50%",
    "limit_timeout_sec": 5
  }
}
```

---

##### 2. Modo `single_evaluation` (Avaliação com Limites Customizados)

> 💡 **Nota:** Requer que o servidor esteja com `SIGE_API_MODE=single_evaluation`. Permite sobrescrever limites de recursos na requisição (que são automaticamente contidos pelos tetos `SIGE_CEILING_*`).

**Requisição (`curl` com limites de memória, CPU e timeout):**
```bash
curl -X POST http://127.0.0.1:8080/execute \
  -H "X-API-Key: SUA_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "language": "c",
    "code": "#include <stdio.h>\nint main() { int a, b; if (scanf(\"%d %d\", &a, &b) == 2) printf(\"%d\\n\", a * b); return 0; }",
    "stdin": "6 7\n",
    "expected_stdout": "42\n",
    "memory_mb": 64,
    "cpu": "80%",
    "timeout_sec": 3,
    "tmp_limit_mb": 32,
    "max_file_size_mb": 5,
    "max_open_files": 128
  }'
```

**Resposta de Sucesso (`PASS`):**
```json
{
  "mode": "single_evaluation",
  "result": "PASS",
  "passed_count": 1,
  "total_count": 1,
  "execution": {
    "stdout": "42\n",
    "stderr": "",
    "duration_ms": 5,
    "memory_peak_bytes": 2097152,
    "exit_code": 0,
    "status": "success",
    "limit_memory_bytes": 67108864,
    "limit_cpu": "80%",
    "limit_timeout_sec": 3
  }
}
```

**Resposta em caso de Discrepância (`FAIL` / `output_mismatch`):**
```json
{
  "mode": "single_evaluation",
  "result": "FAIL",
  "error_type": "output_mismatch",
  "expected": "42\n",
  "actual": "0\n",
  "passed_count": 0,
  "total_count": 1,
  "execution": {
    "stdout": "0\n",
    "stderr": "",
    "duration_ms": 4,
    "exit_code": 0,
    "status": "success"
  }
}
```

---

##### 3. Modo `multi_evaluation` (Bateria de Testes Concorrentes com Limites)

> 💡 **Nota:** Requer que o servidor esteja com `SIGE_API_MODE=multi_evaluation`. Executa até 20 casos de teste em paralelo (em lotes de até 4 sandboxes simultâneos) aplicando os limites customizados informados.

**Requisição (`curl` com bateria de testes):**
```bash
curl -X POST http://127.0.0.1:8080/execute \
  -H "X-API-Key: SUA_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "language": "cpp",
    "code": "#include <iostream>\nusing namespace std;\nint main() { int n; cin >> n; cout << (n % 2 == 0 ? \"PAR\" : \"IMPAR\") << endl; return 0; }",
    "memory_mb": 128,
    "cpu": "100%",
    "timeout_sec": 2,
    "test_cases": [
      { "stdin": "2\n", "expected_stdout": "PAR\n" },
      { "stdin": "7\n", "expected_stdout": "IMPAR\n" },
      { "stdin": "100\n", "expected_stdout": "PAR\n" },
      { "stdin": "0\n", "expected_stdout": "PAR\n" }
    ]
  }'
```

**Resposta com Todos os Testes Aprovados (`PASS`):**
```json
{
  "mode": "multi_evaluation",
  "result": "PASS",
  "passed_count": 4,
  "total_count": 4
}
```

**Resposta com Falha Parcial (aponta a primeira falha ocorrida):**
```json
{
  "mode": "multi_evaluation",
  "result": "FAIL",
  "error_type": "output_mismatch",
  "expected": "PAR\n",
  "actual": "IMPAR\n",
  "failed_test_index": 3,
  "passed_count": 3,
  "total_count": 4,
  "execution": {
    "stdout": "IMPAR\n",
    "stderr": "",
    "duration_ms": 3,
    "exit_code": 0,
    "status": "success"
  }
}
```

---

#### Tabela Completa de Parâmetros da Requisição

| Campo | Tipo | Modos Permitidos | Obrigatório | Descrição |
| :--- | :--- | :--- | :--- | :--- |
| `language` | `string` | Todos | **Sim** | Linguagem do código: `python`, `python3`, `bash`, `sh`, `c`, `cpp`, `c++`. |
| `code` | `string` | Todos | Condicional* | Código-fonte em texto claro (obrigatório se `file_base64` não for enviado). |
| `file_base64` | `string` | Todos | Condicional* | Código-fonte codificado em Base64 (máx. 2 MB). |
| `filename` | `string` | Todos | Não | Nome do arquivo a ser criado no workspace efêmero. |
| `stdin` | `string` | `interpreter`, `single` | Não | Entrada enviada para a `stdin` do processo. |
| `expected_stdout` | `string` | `single_evaluation` | **Sim** (no modo single) | Saída esperada para validação de acerto. |
| `test_cases` | `array` | `multi_evaluation` | **Sim** (no modo multi) | Lista de até 20 objetos `[{ "stdin": "...", "expected_stdout": "..." }]`. |
| `memory_mb` | `int` | `single`, `multi` | Não | Limite de memória RAM em MB (ex: `64`). Limitado pelo teto do servidor. |
| `cpu` | `string` | `single`, `multi` | Não | Cota de CPU (ex: `"50%"`, `"100"` ou `"50000 100000"`). |
| `timeout_sec` | `int` | `single`, `multi` | Não | Tempo máximo de execução em segundos (ex: `3`). |
| `tmp_limit_mb` | `int` | `single`, `multi` | Não | Tamanho do `/tmp` efêmero em tmpfs (ex: `32`). |
| `max_file_size_mb` | `int` | `single`, `multi` | Não | Tamanho máximo de arquivo gravável em disco via `RLIMIT_FSIZE` (ex: `10`). |
| `max_open_files` | `int` | `single`, `multi` | Não | Número máximo de descritores abertos simultâneos via `RLIMIT_NOFILE` (ex: `128`). |

#### Status de Execução (`execution.status`)

| Status | Descrição |
| :--- | :--- |
| `success` | Execução concluída normalmente com `exit_code == 0`. |
| `timeout` | Tempo limite atingido; o cgroup inteiro foi finalizado com `SIGKILL`. |
| `oom` | Limite de memória ultrapassado (*Out of Memory*). |
| `output_limit_exceeded` | Saída (`stdout`/`stderr`) ultrapassou o teto de 1 MB. |
| `file_size_exceeded` | Tentativa de gravação de arquivo além do permitido por `RLIMIT_FSIZE`. |
| `compilation_error` | Falha na compilação do código C/C++ (detalhes retornam em `stderr`). |
| `failed` | Processo encerrou com código de erro ou foi abortado por violação de seccomp. |

---

## ⚙️ Variáveis de Ambiente

Crie um arquivo `.env` na raiz do projeto para ajustar as configurações:

```ini
# ==============================================================================
# SEGURANÇA E AUTENTICAÇÃO
# ==============================================================================
# Chave de acesso da API. Se vazia, uma nova chave será gerada e persistida.
SIGE_API_KEY=

# Define se requisições não autenticadas são aceitas (APENAS para desenvolvimento local)
SIGE_ALLOW_UNAUTHENTICATED=false

# Lista de IPs de proxies confiáveis separados por vírgula (para leitura de X-Real-IP)
SIGE_TRUSTED_PROXIES=

# ==============================================================================
# MODO DE OPERAÇÃO E CONCORRÊNCIA
# ==============================================================================
# Modos: interpreter | single_evaluation | multi_evaluation
SIGE_API_MODE=interpreter

# Concorrência máxima global de sandboxes simultâneos no servidor
SIGE_MAX_CONCURRENT_SANDBOXES=8

# ==============================================================================
# LIMITES PADRÃO POR EXECUÇÃO
# ==============================================================================
SIGE_MEMORY_MB=50
SIGE_CPU=50%
SIGE_TIMEOUT_SEC=5
SIGE_TMP_LIMIT_MB=64
SIGE_MAX_FILE_SIZE_MB=15
SIGE_MAX_OPEN_FILES=256

# ==============================================================================
# TETOS MÁXIMOS DO SERVIDOR (CEILINGS)
# Valores máximos que uma requisição pode solicitar nos modos de avaliação
# ==============================================================================
SIGE_CEILING_MEMORY_MB=512
SIGE_CEILING_CPU_PERCENT=100
SIGE_CEILING_TIMEOUT_SEC=30
SIGE_CEILING_TMP_LIMIT_MB=256
SIGE_CEILING_MAX_FILE_SIZE_MB=64
SIGE_CEILING_MAX_OPEN_FILES=512
```

---

## 📁 Estrutura de Pastas

```text
sige/
├── cmd/                          # Pontos de entrada CLI e comandos Cobra
│   ├── api.go                    # Inicialização do servidor HTTP REST
│   ├── run_task.go               # Comando CLI 'run-task' para execução avulsa
│   ├── internal_launch.go        # Executor interno de inicialização do sandbox
│   ├── verify_cgroups.go         # Diagnóstico de versão e suporte a cgroups
│   └── bootstrap.go              # Lógica de montagem e bootstrap do isolamento
├── internal/                     # Pacotes internos da aplicação
│   ├── api/                      # Servidor HTTP, roteamento, validação e middlewares
│   │   ├── engines.go            # Motores de avaliação (single, multi, interpreter)
│   │   ├── server.go             # Handlers HTTP, rate limiter e autenticação
│   │   ├── types.go              # Modelos de dados e contratos da API
│   │   ├── validators.go         # Validação semântica e sanitização de requisições
│   │   └── workspace.go          # Gerenciamento de diretórios temporários por execução
│   ├── sandbox/                  # Motor de contenção no kernel
│   │   ├── namespace.go          # Configuração de namespaces Linux e pivot_root
│   │   ├── seccomp.go            # Filtros BPF e regras libseccomp
│   │   ├── baseline_seccomp.go   # Definição das syscalls essenciais permitidas
│   │   ├── executor.go           # Execução de comandos com limites rlimits
│   │   └── config.go             # Resolução de limites e parâmetros de execução
│   ├── cgroups/                  # Gerenciador cgroups v2 e delegação de controladores
│   └── constants/                # Constantes e limites do sistema
├── test/                         # Bateria de testes
│   ├── real_cases/               # Suíte adversarial com 26 cenários de ataque reais
│   └── run_real_cases_test.go    # Testes de integração automatizados
├── Dockerfile                    # Multi-stage build com libseccomp e runtime Debian
├── docker-compose.yml            # Orquestração do serviço com confinamento seguro
└── go.mod                        # Módulos e dependências em Go
```

---

## 🧪 Testes e Segurança

### Testes Unitários

Execute a suíte de testes unitários:

```bash
go test -v ./...
```
*(Testes que dependem de privilégios de kernel detectam a ausência de capacidades e se auto-pulam com segurança).*

### Suíte Adversarial de Segurança (`real_cases`)

O projeto inclui uma bateria completa com **26 testes adversariais** em `test/real_cases/`, testando o comportamento contra os seguintes ataques:
- [x] **Fork Bomb:** Esgotamento de PIDs neutralizado por `cgroup pids.max`.
- [x] **Consumo Excessivo de RAM:** Alocações em loop neutralizadas por `cgroup memory.max`.
- [x] **Escape de Filesystem:** Tentativas de escrita em `/bin` e `/usr` bloqueadas por rootfs read-only.
- [x] **Violação de Syscall:** Tentativa de chamada a `unshare()` ou `ptrace()` terminada pelo filtro `seccomp BPF`.
- [x] **Esgotamento de Descritores & Disco:** Bloqueado via `RLIMIT_NOFILE` e `RLIMIT_FSIZE`.
- [x] **Ataques de Rede & SSRF:** Tentativas de conexão externa bloqueadas por `CLONE_NEWNET`.

Para rodar os testes de integração adversariais contra o container ativo:

```bash
go test -v -run TestRealCases ./test/
```

### Análise de Vulnerabilidades

Recomenda-se executar o `govulncheck` antes de submeter alterações:

```bash
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

---

## 🤝 Como Contribuir

Contribuições são muito bem-vindas! Siga estas etapas para contribuir com o projeto:

1. **Faça um Fork** do projeto no GitHub.
2. **Crie uma branch** para sua feature ou correção:
   ```bash
   git checkout -b feature/minha-nova-funcionalidade
   ```
3. **Escreva seu código** seguindo as convenções idiomáticas do Go (`gofmt`, `go vet`, boas práticas de segurança).
4. **Adicione testes** para novas funcionalidades e certifique-se de que a suíte existente continua passando:
   ```bash
   go test ./...
   ```
5. **Faça o Commit** das suas alterações com mensagens claras e descritivas:
   ```bash
   git commit -m "feat(sandbox): adiciona suporte a limite customizado de pids"
   ```
6. **Envie para o seu repositório remoto:**
   ```bash
   git push origin feature/minha-nova-funcionalidade
   ```
7. **Abra um Pull Request** detalhando as mudanças propostas, motivação e resultados dos testes.

---

## 📄 Licença e Autor

Distribuído sob a licença **Apache 2.0**. Consulte o arquivo [LICENSE](LICENSE) para mais detalhes.

Desenvolvido por **[Geraldo Filho](https://github.com/Geraldoaf)** como parte de pesquisa e Trabalho de Conclusão de Curso (TCC) em Engenharia de Software / Ciência da Computação, focado em segurança ofensiva e isolamento de sistemas no Linux.

---

<p align="center">
  <sub>Construído com dedicação à segurança e engenharia de baixo nível. Dúvidas ou sugestões? Abra uma <a href="https://github.com/Geraldoaf/sige/issues">issue</a>!</sub>
</p>
