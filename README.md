# SIGE - Sistema de Isolamento e Gerenciamento de Execução

Este repositório contém a implementação do **SIGE: Sandbox de Isolamento Seguro e Concorrente para Execução de Código Não Confiável**, desenvolvida em Go como projeto de TCC.

## Recurso de Destaque
- **Isolamento de Recursos:** Limites de RAM (`memory.max`), CPU (`cpu.max`) e PIDs (`pids.max = 50`) via **cgroups v2**.
- **Isolamento de Processos e Rede:** Linux Namespaces (`PID`, `NET`, `UTS`, `IPC`, `NS`).
- **Sistema de Arquivos:** Montagens Read-Only com `pivot_root` e diretório `/tmp` em `tmpfs`.
- **Filtro de Chamadas de Sistema:** **Seccomp BPF** via `libseccomp` com intercepção `ActKillProcess`.
- **Encerramento Atômico:** Suporte a `cgroup.kill` (Kernel 5.14+).
- **Concorrência:** Servidor HTTP concorrente e avaliação paralela de múltiplos casos de teste com **Goroutines** e `sync.WaitGroup`.

## Estrutura do Repositório

- `cmd/`: CLI da aplicação (subcomandos `api`, `run-task`, `init-config`, `cgroups-version`).
- `internal/sandbox/`: Motor de isolamento (cgroups, namespaces, seccomp, rlimits).
- `internal/api/`: Servidor REST HTTP e avaliadores de submissões (`interpreter`, `single_evaluation`, `multi_evaluation`).
- `internal/cgroups/`: Integração com a hierarquia unificada v2 do Linux.
- `config.json`: Arquivo de configuração de limites padrão.
- `docker-compose.yml`: Containerização para implantação.

## Instalação e Execução

### 1. Inicializar Configurações
```bash
go build -o sige main.go
./sige init-config
```

### 2. Rodar Sandbox via CLI
```bash
./sige run-task --mem 50 --timeout 3 -- python3 -c "print('Hello Sandbox')"
```

### 3. Rodar Servidor HTTP da API
```bash
./sige api -p 8080
```

### 4. Executar via Docker
```bash
docker-compose up --build
```
