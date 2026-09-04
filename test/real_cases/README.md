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
| 27 | `27_symlink_escape.py` | Tentativa de escape e quebra de sandbox via symlinks |
| 28 | `28_signal_trap.py` | Captura de todos os sinais e loop infinito (força timeout/SIGKILL) |
| 29 | `29_tmpfs_exhaustion.py` | Esgotamento de espaço em disco no `/tmp` (quota tmpfs) |
| 30 | `30_ipc_keyring_access.c` | Isolamento de IPC SysV e bloqueio de keyctl |
| 31 | `31_zombie_exhaustion.c` | Criação massiva de processos zumbis (limite pids.max) |
| 32 | `32_coredump_trigger.c` | Geração de core dump e teste de `RLIMIT_CORE` |
| 33 | `33_preprocessor_bomb.c` | Ataque de expansão macro exponencial no compilador C |
| 34 | `34_procfs_probing.py` | Acesso a entradas sensíveis do `/proc` (`kcore`, `kallsyms`, etc.) |
| 35 | `35_inherited_fd_leak.py` | Varredura de descritores de arquivo (FDs 3-100) herdados |
| 36 | `36_binary_null_flood.py` | Emissão de bytes nulos binários e UTF-8 corrompido para API |
| 37 | `37_cve_2024_28185_symlink_write.py` | Classe da CVE-2024-28185: symlink desviando escrita do host |
| 38 | `38_cve_2024_28189_symlink_chown.py` | Classe da CVE-2024-28189: symlink desviando `chown` do host |
| 39 | `39_cve_2024_29021_ssrf.py` | Classe da CVE-2024-29021: SSRF para alvos internos |

Todos os casos podem ser submetidos via POST `/execute` no servidor da API.

## Rastreabilidade: CVEs do Judge0

Os casos 37–39 existem para dar evidência executável — e não apenas
argumentativa — de que o SIGE não é suscetível às classes de ataque das três
CVEs de severidade máxima (CVSS 10.0) divulgadas no Judge0 em abril de 2024.

| CVE | Classe de ataque | Caso | Por que o SIGE resiste |
| :--- | :--- | :--- | :--- |
| CVE-2024-28185 | Symlink plantado pelo código não confiável desvia uma **escrita** do host para fora do sandbox | 37 | `/workspace` é somente-leitura durante a execução, e a escrita do host ocorre **antes** de qualquer código não confiável rodar |
| CVE-2024-28189 | Mesma técnica aplicada ao **`chown`** que o host executava | 38 | `chown` não consta da allowlist de syscalls: a operação é inalcançável, não apenas negada. Nenhum `chown` do servidor incide sobre caminho derivado de entrada do usuário |
| CVE-2024-29021 | **SSRF** via recurso de callback do servidor | 39 | O SIGE não possui callback, webhook ou qualquer requisição HTTP de saída; e o namespace de rede do sandbox não tem interface |

**Nota de precisão terminológica.** O SIGE não "corrige" nem "bloqueia" essas
CVEs — elas são falhas no código do Judge0, não técnicas genéricas. O que os
casos demonstram é que o SIGE **não é suscetível às classes de ataque** que
elas representam. Ao citar estes resultados, prefira essa formulação.

**Verificação complementar do lado do host.** Os casos 37 e 38 observam o
sistema de dentro do sandbox. Para fechar a evidência, confirme de fora que
nenhum arquivo apareceu onde os links apontavam:

```bash
docker compose exec sige-api ls -la /etc/sige_pwned_37 2>&1   # esperado: No such file
docker compose exec sige-api stat -c '%U:%G %a' /etc/passwd   # esperado: root:root 644, inalterado
```

O caso 38 termina com o processo morto pelo seccomp, então sua saída para na
linha `Action: chown ...` — isso é o resultado esperado, conforme a convenção
descrita acima.
