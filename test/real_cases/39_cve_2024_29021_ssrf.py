import socket

# =====================================================================
# [TEST 39] Classe de ataque da CVE-2024-29021 (Judge0, CVSS 10.0)
#
# MECANISMO DA VULNERABILIDADE ORIGINAL
# O Judge0 oferece um recurso de callback: o cliente informa uma URL e o
# SERVIDOR a busca ao terminar a submissão. Isso é um SSRF por construção
# — o atacante faz o servidor emitir requisições para destinos que ele
# próprio não alcança: serviços internos, o endpoint de metadados da nuvem
# (169.254.169.254), o socket do Docker. Combinado com as outras falhas,
# levava a comprometimento do host.
#
# O QUE ESTE TESTE VERIFICA NO SIGE
# A superfície tem dois lados, e só um é observável de dentro do sandbox:
#
#   1. Lado do servidor (NÃO testável daqui, verificável por inspeção):
#      o SIGE não possui callback, webhook, nem qualquer requisição HTTP
#      de saída. Uma varredura por http.Get/http.Post/http.Client no
#      código não retorna nenhuma ocorrência. Não existindo o recurso,
#      não existe a superfície.
#
#   2. Lado do sandbox (testado aqui): mesmo que o código não confiável
#      tente ele mesmo alcançar destinos internos, o namespace de rede
#      próprio e sem interface torna qualquer destino inalcançável.
#
# Este caso complementa o 08 mirando especificamente nos alvos clássicos
# de SSRF, não em um destino externo genérico.
# =====================================================================

print("=== [TEST 39] CVE-2024-29021 class: SSRF to internal targets ===", flush=True)
print("Action: Attempting to reach the destinations an SSRF would target...", flush=True)

alvos = [
    ("169.254.169.254", 80, "metadados de nuvem (AWS/GCP/Azure)"),
    ("127.0.0.1", 8080, "a propria API do SIGE"),
    ("127.0.0.1", 6379, "Redis interno"),
    ("127.0.0.1", 5432, "PostgreSQL interno"),
    ("10.0.0.1", 80, "rede privada RFC1918"),
    ("172.17.0.1", 80, "gateway padrao do Docker (host)"),
]

bloqueados = 0
for host, porta, descricao in alvos:
    try:
        s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        s.settimeout(2)
        s.connect((host, porta))
        print(f"Result: UNEXPECTED - conectou em {host}:{porta} ({descricao})", flush=True)
        s.close()
    except OSError as e:
        bloqueados += 1
        print(f"OK - {host}:{porta} inalcancavel ({e.strerror}) - {descricao}", flush=True)

print(f"Destinos bloqueados: {bloqueados}/{len(alvos)}", flush=True)

# Resolução de nomes também não deve funcionar: sem interface de rede, não
# há como falar com servidor DNS algum.
try:
    ip = socket.gethostbyname("metadata.google.internal")
    print(f"Result: UNEXPECTED - resolveu DNS para {ip}", flush=True)
except OSError as e:
    print(f"OK - resolucao de DNS indisponivel ({type(e).__name__})", flush=True)

# O socket do Docker é o alvo de escape mais valioso quando exposto.
import os
for caminho in ("/var/run/docker.sock", "/run/docker.sock"):
    if os.path.exists(caminho):
        print(f"Result: UNEXPECTED - {caminho} visivel dentro do sandbox!", flush=True)
    else:
        print(f"OK - {caminho} ausente do sandbox", flush=True)

print("=== END TEST 39 ===", flush=True)
