# test/real_cases

Payloads pra submeter em `/execute` e observar o comportamento do sandbox.
Cada arquivo é independente — não precisa de bibliotecas externas além do
que já vem instalado no container (`python3`, `gcc`, `g++`).

## Convenção

- Numeração sequencial, um caso por arquivo.
- Cabeçalho `=== [TEST NN] ... ===`, seguido de `Action: ...` descrevendo o
  que o teste tenta fazer, e `Result: ...` com o que de fato aconteceu.
- Linguagem inferida pela extensão: `.py` → `language: "python"`, `.c` →
  `"c"`, `.cpp` → `"cpp"`, `.sh` → `"bash"`.

## Como interpretar testes bloqueados por seccomp (09, 10, 11)

`ActKillProcess` mata o processo *na hora* da chamada — não é um erro
capturável. Isso significa que, se o bloqueio funcionou, o `stdout` da
execução **para exatamente depois da linha "Action: ..."**, e o
`exit_code`/`status` da resposta da API indica término por sinal (não um
retorno normal, e não `success`). As linhas de `Result: ...` desses arquivos
só aparecem se o bloqueio **falhou** — leia-as como "isto não deveria
aparecer".

Já os testes de limite de recursos (memória, arquivos, cgroup pids, output)
usam mecanismos que retornam erro normal do SO (`OSError`, `errno`), então
essas capturam e imprimem `Result: BLOCKED - ...` normalmente.

## Índice

| # | Arquivo | O que verifica |
|---|---|---|
| 01 | `01_quicksort.py` | Execução correta, caso feliz |
| 02 | `02_oom_alloc.py` | Limite de memória (cgroup) |
| 03 | `03_infinite_loop.py` | Timeout |
| 04 | `04_file_overflow.py` | `RLIMIT_FSIZE` / `max_file_size_mb` |
| 05 | `05_write_rootfs.py` | Rootfs read-only (`/bin`) |
| 06 | `06_seccomp_blocked.py` | `unshare()` bloqueado (via ctypes) |
| 07 | `07_fork_bomb.py` | `cgroup pids.max` |
| 08 | `08_network_isolation.py` | `CLONE_NEWNET` sem rota/interface |
| 09 | `09_clone_namespace_flags.c` | Regra condicional nova: `clone()` com `CLONE_NEWUSER` |
| 10 | `10_ptrace_blocked.c` | `ptrace()` bloqueado |
| 11 | `11_raw_mount_blocked.c` | `mount()` bloqueado mesmo sem capability |
| 12 | `12_capability_verification.c` | Confirma `CapEff`/`CapPrm` zerados, `NoNewPrivs: 1`, UID/GID 65534 |
| 13 | `13_orphan_process.sh` | Reaping de processo órfão (morte do namespace de PID) — checar de fora do container |
| 14 | `14_output_flood.py` | Limite de 1MB de stdout (interpretado) |
| 15 | `15_output_flood.cpp` | Mesmo limite, binário compilado |
| 16 | `16_open_files_exhaustion.py` | `RLIMIT_NOFILE` |
| 17 | `17_env_sanitization.py` | Variáveis de ambiente do host não vazam pro sandbox |
| 18 | `18_compile_leak_etc_passwd.c` | Compilação sandboxada (achado mais crítico da auditoria) |
| 19 | `19_compile_leak_etc_passwd.cpp` | Mesmo teste, g++ |
| 20 | `20_compile_leak_etc_shadow.c` | Mesmo ataque contra arquivo sem permissão de leitura (nobody) |
| 21 | `21_compile_resource_bomb.cpp` | Limite de memória/tempo da própria compilação |
| 22 | `22_clone3_blocked.c` | `clone3` negada — retorna ENOSYS (e não kill) para permitir o fallback da glibc |
| 23 | `23_write_sandbox_root.py` | Raiz `/` do sandbox somente-leitura (com `/tmp` ainda gravável como controle) |
| 24 | `24_dev_nodes.py` | `/dev` mínimo disponível (`null`, `zero`, `random`, `urandom`, symlinks) e somente-leitura |
| 25 | `25_read_application.py` | Binários, config e segredos do SIGE inalcançáveis de dentro do sandbox (equivalente ao fix v1.4.0 do Judge0) |
| 26 | `26_concurrency.py` | Threads, `multiprocessing` e `/dev/shm` funcionam (protege contra a regressão do `clone3`) |

Os testes 02–22 ainda não estão todos plugados em `run_real_cases_test.go`
(hoje só o 01 roda via `go test`) — dá pra rodar manualmente com `curl` contra
uma instância local, ou posso estender o harness Go pra rodar o lote inteiro
e checar o `status`/`result` esperado de cada um automaticamente.
