import os

# =====================================================================
# [TEST 38] Classe de ataque da CVE-2024-28189 (Judge0, CVSS 10.0)
#
# MECANISMO DA VULNERABILIDADE ORIGINAL
# A CVE-2024-28189 é o bypass do patch da CVE-2024-28185 (ver caso 37).
# Depois de a ESCRITA do host ter sido protegida, sobrou uma segunda
# operação que o host executava sobre arquivo controlado pelo atacante
# dentro do sandbox: o `chown`. Como chown segue links simbólicos, um link
# apontando para fora fazia o host trocar o dono de um arquivo arbitrário
# do sistema — escape novamente.
#
# A lição da dupla 28185/28189: não basta proteger a escrita. QUALQUER
# operação do host sobre caminho influenciável pelo código não confiável é
# vetor — escrita, chown, chmod, rename, remoção.
#
# COMO LER O RESULTADO DESTE TESTE
# A saída termina na linha "Action: chown ...". Isso é o resultado
# ESPERADO, não uma falha do teste: `chown` não consta da allowlist de
# syscalls (ver allowedSyscalls em internal/sandbox/seccomp.go), então a
# chamada mata o processo via SECCOMP_RET_KILL_PROCESS. A operação que o
# host do Judge0 executava é, no SIGE, inalcançável pelo código do usuário
# — não é negada por permissão, é negada na fronteira da syscall.
#
# Cada print usa flush=True porque, ao ser morto por sinal, o processo não
# esvazia o buffer de saída — sem o flush, todo o diagnóstico se perderia.
#
# LADO DO SERVIDOR (não observável daqui, verificável por inspeção)
# O SIGE não executa chown sobre nenhum caminho derivado de entrada do
# usuário. Os os.Chown do projeto incidem apenas sobre caminhos fixos da
# hierarquia de cgroups (cmd/bootstrap.go), e o os.Chmod incide sobre o
# diretório de workspace, cujo nome é gerado pelo servidor
# (exec-<timestamp>-<pid>), nunca pelo cliente.
# =====================================================================

print("=== [TEST 38] CVE-2024-28189 class: symlink redirects host chown ===", flush=True)
print(f"Contexto: uid={os.getuid()} gid={os.getgid()}", flush=True)

# Link auxiliar apontando para fora do diretório de trabalho — é o que o
# atacante plantaria para desviar a operação do host.
fora = "/etc/passwd"
link = "/tmp/link_chown_38"
try:
    if os.path.lexists(link):
        os.remove(link)
    os.symlink(fora, link)
    print(f"OK - link auxiliar plantado: {link} -> {fora}", flush=True)
except OSError as e:
    print(f"OK - nem o link auxiliar pode ser criado ({type(e).__name__})", flush=True)
    link = None

# chmod ESTÁ na allowlist, então retorna erro capturável: demonstra a
# negação em nível de permissão (o processo é nobody, sem capabilities).
for alvo in ("/etc/passwd", "/workspace", "/"):
    try:
        os.chmod(alvo, 0o777)
        print(f"Result: UNEXPECTED - chmod 777 aplicado em {alvo}!", flush=True)
    except OSError as e:
        print(f"OK - chmod em {alvo} negado ({type(e).__name__}: {e.strerror})", flush=True)

if link:
    try:
        os.chmod(link, 0o777)
        print("Result: UNEXPECTED - chmod atraves do symlink funcionou!", flush=True)
    except OSError as e:
        print(f"OK - chmod atraves do symlink negado ({type(e).__name__}: {e.strerror})", flush=True)

# A partir daqui o processo deve MORRER. chown não está na allowlist.
print("Action: chown atraves do symlink — esperado: processo morto pelo seccomp", flush=True)
print("        (se a proxima linha aparecer, a syscall NAO foi bloqueada)", flush=True)

os.chown(link if link else fora, 0, 0)

print("Result: UNEXPECTED - chown executou; syscall alcancavel pelo codigo do usuario!", flush=True)
print("=== END TEST 38 ===", flush=True)
