import os

# =====================================================================
# [TEST 37] Classe de ataque da CVE-2024-28185 (Judge0, CVSS 10.0)
#
# MECANISMO DA VULNERABILIDADE ORIGINAL
# No Judge0, o processo do HOST escrevia um arquivo (run_script) para
# DENTRO do diretório do sandbox. Como o código não confiável conseguia
# escrever nesse mesmo diretório, bastava pré-criar ali um link simbólico
# apontando para fora: a escrita do host seguia o link e caía no sistema
# de arquivos do host, resultando em escrita arbitrária e execução de
# código fora do sandbox.
#
# Repare na DIREÇÃO do ataque: não é o sandbox lendo para fora (isso é o
# caso 27). É o HOST escrevendo para fora, redirecionado por um link que o
# sandbox plantou.
#
# O QUE ESTE TESTE VERIFICA NO SIGE
# O SIGE escreve, do lado do host, nestes caminhos (ver
# internal/api/workspace.go):
#   - /workspace/<filename>   → o código-fonte, via os.WriteFile
#   - /workspace/solution     → o binário, produzido pela compilação
#   - /workspace              → removido no fim, via os.RemoveAll
#
# A defesa do SIGE é estrutural e tem duas partes:
#   1. /workspace é montado SOMENTE-LEITURA durante a execução, então o
#      código do usuário não consegue plantar link nenhum ali;
#   2. a escrita do host acontece ANTES de qualquer código não confiável
#      rodar, então não existe a janela que o Judge0 tinha.
#
# Este teste cobre a parte (1), que é o que se pode observar de dentro do
# sandbox. A parte (2) é uma propriedade de ordenação, verificável por
# inspeção de código e pela checagem externa descrita no README.
# =====================================================================

print("=== [TEST 37] CVE-2024-28185 class: symlink redirects host write ===")
print("Action: Planting symlinks where the host writes, pointing outside the sandbox...")

# Nomes que o host efetivamente toca no workspace.
alvos_do_host = ["solution", "solution.py", "solution.c", "solution.cpp", "solution.sh"]

# Destinos fora do sandbox. Se algum destes aparecer no host depois da
# execução, houve escape.
fora = "/etc/sige_pwned_37"

bloqueados = 0
for alvo in alvos_do_host:
    caminho = f"/workspace/{alvo}"
    try:
        # Se o arquivo já existe (é o caso do fonte submetido), tenta
        # substituí-lo por um link — foi assim que o Judge0 caiu.
        if os.path.lexists(caminho):
            os.remove(caminho)
        os.symlink(fora, caminho)
        print(f"Result: UNEXPECTED - planted symlink at {caminho} -> {fora}")
    except OSError as e:
        bloqueados += 1
        print(f"OK - {caminho} protegido ({type(e).__name__}: {e.strerror})")

# Criar arquivo novo no workspace também deve falhar.
try:
    os.symlink(fora, "/workspace/arquivo_novo_37")
    print("Result: UNEXPECTED - criou symlink novo no /workspace!")
except OSError as e:
    bloqueados += 1
    print(f"OK - criação de symlink novo bloqueada ({type(e).__name__})")

print(f"Tentativas bloqueadas: {bloqueados}/{len(alvos_do_host) + 1}")

# Controle: nas áreas graváveis o symlink É criado — mas resolve dentro do
# rootfs pivotado, então não alcança o host. Isso mostra que a proteção vem
# do isolamento, não de o sistema proibir symlinks em geral.
print("Action: Same attempt in writable areas (/tmp, /dev/shm)...")
for base in ("/tmp", "/dev/shm"):
    link = f"{base}/link_37"
    try:
        if os.path.lexists(link):
            os.remove(link)
        os.symlink("/etc/passwd", link)
        destino = os.path.realpath(link)
        primeira = open(link).readline().strip()
        print(f"OK - {link} criado, resolve para {destino}")
        print(f"     conteudo: {primeira[:60]}")
        print("     -> e o /etc/passwd DO SANDBOX (rootfs pivotado), nao do host")
    except OSError as e:
        print(f"OK - {link} bloqueado ({type(e).__name__})")

print("=== END TEST 37 ===")
