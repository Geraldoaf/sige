# SIGE - Sistema de Isolamento e Gerenciamento de Execução

Este repositório contém a implementação do **SIGE: Sandbox de Isolamento Seguro e Concorrente para Execução de Código Não Confiável**, desenvolvida em Go como projeto de TCC.

## Recurso de Destaque
- **Isolamento de Recursos:** Limites de RAM (`memory.max`), CPU (`cpu.max`) e PIDs (`pids.max = 50`) via **cgroups v2**.
- **Isolamento de Processos e Rede:** Linux Namespaces (`PID`, `NET`, `UTS`, `IPC`, `NS`).
- **Sistema de Arquivos:** Montagens Read-Only com `pivot_root` e diretório `/tmp` em `tmpfs`.
- **Filtro de Chamadas de Sistema:** **Seccomp BPF** via `libseccomp` com intercepção `ActKillProcess`.
- **Suporte Multi-Linguagem:** Linguagens interpretadas (`python`, `bash`) e compiladas (`c`, `cpp` / `c++` via `gcc`/`g++`).
- **Encerramento Atômico:** Suporte a `cgroup.kill` (Kernel 5.14+).
- **Concorrência:** Servidor HTTP concorrente e avaliação paralela de múltiplos casos de teste com **Goroutines** e `sync.WaitGroup`.

---

## Como Funciona a API do SIGE

A API HTTP do SIGE expõe o endpoint `POST /execute` para receber submissões de código, compilá-las (se necessário) e executá-las em instâncias isoladas da sandbox.

### Modos de Operação da API (`api_mode`)

O modo de operação da API é definido no arquivo `config.json` através do campo `"api_mode"`. Existem 3 modos disponíveis:

1. **`interpreter` (Padrão):**
   - Executa a submissão e retorna o resultado direto do processo (`stdout`, `stderr`, tempo de CPU, consumo de memória, código de saída e status).
   - Não realiza comparação de saída nem aceita parâmetros de correção automatizada (`expected_stdout` ou `test_cases`).

2. **`single_evaluation`:**
   - Avalia a submissão contra 1 caso de teste fornecido.
   - Compara o `stdout` gerado com o `expected_stdout`. Retorna `PASS` se coincidirem (ignorando espaços nas extremidades) ou `FAIL` com `error_type: "output_mismatch"`.
   - Permite que a requisição sobrescreva limites temporários de recursos (`memory_mb`, `timeout_sec`, etc.).

3. **`multi_evaluation`:**
   - Avalia a submissão contra uma lista de múltiplos casos de teste (`test_cases`).
   - Cada caso de teste é executado em **paralelo** em Goroutines separadas, cada uma em sua própria sandbox isolada.
   - Retorna o total de testes aprovados (`passed_count`), total executado (`total_count`) e detalhes da primeira falha encontrada.

---

## Estrutura da Requisição POST (`POST /execute`)

### Endpoint
```http
POST /execute
Content-Type: application/json
```

### Campos do Payload JSON (`ExecuteRequest`)

| Campo | Tipo | Obrigatório | Descrição |
| :--- | :--- | :--- | :--- |
| `language` | `string` | **Sim** | Linguagem do código: `"python"` / `"python3"`, `"bash"` / `"sh"`, `"c"`, `"cpp"` / `"c++"`. |
| `code` | `string` | *Condicional* | Código-fonte em texto plano. (Obrigatório caso `file_base64` não seja enviado). |
| `file_base64` | `string` | *Condicional* | Código-fonte codificado em Base64. |
| `filename` | `string` | Não | Nome customizado do arquivo (ex: `"main.c"`). |
| `stdin` | `string` | Não | Entrada padrão enviada ao programa durante a execução. |
| `expected_stdout` | `string` | Modo `single_evaluation` | Saída esperada para comparação individual. |
| `test_cases` | `array` | Modo `multi_evaluation` | Lista de objetos `{ "stdin": "...", "expected_stdout": "..." }`. |
| `memory_mb` | `int` | Não | Limite de memória RAM em MB (sobrescreve o padrão se no modo de avaliação). |
| `cpu` | `string` | Não | Limite de CPU (ex: `"10"` para 10% ou em formato cgroup `"10000 100000"`). |
| `timeout_sec` | `int` | Não | Limite de tempo de execução em segundos. |
| `tmp_limit_mb` | `int` | Não | Limite de espaço em MB para a pasta `/tmp` em `tmpfs`. |
| `max_file_size_mb` | `int` | Não | Tamanho máximo permitido para escrita de arquivos (MB). |
| `max_open_files` | `int` | Não | Limite de descritores de arquivos/soquetes abertos. |

---

## Exemplos de Uso da API

### 1. Execução Simples em Python (Modo `interpreter`)

#### Requisição:
```bash
curl -X POST http://localhost:8080/execute \
  -H "Content-Type: application/json" \
  -d '{
    "language": "python",
    "code": "print(\"Olá do SIGE!\")"
  }'
```

#### Resposta:
```json
{
  "mode": "interpreter",
  "result": "completed",
  "passed_count": 1,
  "total_count": 1,
  "execution": {
    "stdout": "Olá do SIGE!\n",
    "stderr": "",
    "duration_ms": 15,
    "cpu_user_time_us": 12000,
    "cpu_system_time_us": 3000,
    "memory_peak_bytes": 8388608,
    "exit_code": 0,
    "status": "success",
    "limit_memory_bytes": 52428800,
    "limit_timeout_sec": 5
  }
}
```

---

### 2. Avaliação de Código em C (Modo `single_evaluation`)

#### Requisição:
```bash
curl -X POST http://localhost:8080/execute \
  -H "Content-Type: application/json" \
  -d '{
    "language": "c",
    "code": "#include <stdio.h>\nint main() { int a, b; scanf(\"%d %d\", &a, &b); printf(\"%d\\n\", a + b); return 0; }",
    "stdin": "10 20",
    "expected_stdout": "30"
  }'
```

#### Resposta (Aprovado - `PASS`):
```json
{
  "mode": "single_evaluation",
  "result": "PASS",
  "passed_count": 1,
  "total_count": 1,
  "execution": {
    "stdout": "30\n",
    "stderr": "",
    "duration_ms": 2,
    "exit_code": 0,
    "status": "success"
  }
}
```

---

### 3. Avaliação Paralela em C++ com Múltiplos Casos de Teste (Modo `multi_evaluation`)

#### Requisição:
```bash
curl -X POST http://localhost:8080/execute \
  -H "Content-Type: application/json" \
  -d '{
    "language": "cpp",
    "code": "#include <iostream>\nusing namespace std;\nint main() { int n; cin >> n; cout << n*n << endl; return 0; }",
    "test_cases": [
      { "stdin": "2", "expected_stdout": "4" },
      { "stdin": "5", "expected_stdout": "25" },
      { "stdin": "10", "expected_stdout": "100" }
    ]
  }'
```

#### Resposta:
```json
{
  "mode": "multi_evaluation",
  "result": "PASS",
  "passed_count": 3,
  "total_count": 3
}
```

---

### 4. Retorno em Caso de Erro de Compilação (`compilation_error`)

Caso o código fornecido em C ou C++ contenha erros de sintaxe, o SIGE captura o `stderr` do compilador e retorna HTTP 200 com a explicação:

#### Resposta:
```json
{
  "mode": "interpreter",
  "result": "failed",
  "error_type": "compilation_error",
  "passed_count": 0,
  "total_count": 1,
  "execution": {
    "stdout": "",
    "stderr": "solution.c: In function 'main':\nsolution.c:3:5: error: expected ';' before 'return'\n",
    "status": "compilation_error"
  }
}
```

---

## Estrutura do Repositório

- `cmd/`: CLI da aplicação (subcomandos `api`, `run-task`, `init-config`, `cgroups-version`).
- `internal/sandbox/`: Motor de isolamento (cgroups v2, namespaces, seccomp BPF, rlimits).
- `internal/api/`: Servidor REST HTTP, manipulador de workspace e avaliadores de submissões (`interpreter`, `single_evaluation`, `multi_evaluation`).
- `internal/cgroups/`: Integração com a hierarquia unificada v2 do Linux.
- `config.json`: Arquivo de configuração de limites padrão e modo da API.
- `docker-compose.yml`: Containerização para implantação.

---

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
