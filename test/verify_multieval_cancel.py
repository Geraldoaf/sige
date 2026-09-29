#!/usr/bin/env python3
"""
verify_multieval_cancel.py

Verifica a correção do bug 2.2 (multi_evaluation context propagation e cancelamento).
Inicia uma requisição de multi_evaluation demorada, aborta a conexão TCP no meio
e valida se a capacidade ativa do servidor é liberada rapidamente.
"""

import sys
import time
import json
import socket
import urllib.request

BASE_HOST = "127.0.0.1"
BASE_PORT = 8080
API_KEY = sys.argv[1] if len(sys.argv) > 1 else ""

def get_server_info():
    try:
        url = f"http://{BASE_HOST}:{BASE_PORT}/capacity"
        req = urllib.request.Request(url)
        if API_KEY:
            req.add_header("X-API-Key", API_KEY)
        with urllib.request.urlopen(req, timeout=3) as resp:
            data = json.loads(resp.read().decode())
            return {
                "active": data.get("active_sandboxes", data.get("active", 0)),
                "mode": data.get("api_mode", "interpreter")
            }
    except Exception as e:
        return {"active": 0, "mode": "interpreter"}

def get_active_sandboxes():
    return get_server_info()["active"]

def main():
    print(f"=== [VERIFICAÇÃO API] Cancelamento de execução por desconexão de cliente ===")
    
    info = get_server_info()
    initial_active = info["active"]
    api_mode = info["mode"]
    print(f"[*] Modo da API: {api_mode}, Sandboxes ativos antes do teste: {initial_active}")

    if api_mode == "multi_evaluation":
        test_cases = [{"stdin": f"test_{i}\n", "expected_stdout": f"out_{i}\n"} for i in range(10)]
        code = "import time, sys\nline = sys.stdin.read().strip()\ntime.sleep(5)\nprint(f'out_{line}')\n"
        payload_data = {
            "language": "python",
            "code": code,
            "test_cases": test_cases
        }
    else:
        # Modo interpreter: uma execução longa de 10s
        payload_data = {
            "language": "python",
            "code": "import time\nprint('starting sleep')\ntime.sleep(10)\nprint('finished sleep')\n"
        }

    payload = json.dumps(payload_data)

    http_request = (
        f"POST /execute HTTP/1.1\r\n"
        f"Host: {BASE_HOST}:{BASE_PORT}\r\n"
        f"Content-Type: application/json\r\n"
        f"Content-Length: {len(payload)}\r\n"
    )
    if API_KEY:
        http_request += f"X-API-Key: {API_KEY}\r\n"
    http_request += f"Connection: close\r\n\r\n{payload}"

    print("[*] Abrindo socket TCP e enviando requisição HTTP longa de multi_evaluation...")
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.connect((BASE_HOST, BASE_PORT))
    s.sendall(http_request.encode())

    # Aguarda 1.5s para que os primeiros sandboxes iniciem a execução
    time.sleep(1.5)
    during_active = get_active_sandboxes()
    print(f"[*] Sandboxes ativos após início: {during_active}")

    print("[*] Abortando brutalmente a conexão TCP do cliente (fechando socket)...")
    s.close()

    # O servidor deve detectar o cancelamento do context HTTP e encerrar os processos
    print("[*] Monitorando se os sandboxes encerram rapidamente após desconexão...")
    cancelled_ok = False
    for attempt in range(8):
        time.sleep(0.8)
        current_active = get_active_sandboxes()
        print(f"    [T+{attempt*0.8:.1f}s] Sandboxes ativos: {current_active}")
        if current_active == 0:
            cancelled_ok = True
            break

    if cancelled_ok:
        print("\n>>> SUCESSO: Processos cancelados com sucesso após desconexão do cliente! <<<")
        return 0
    else:
        print("\n>>> FALHA: Sandboxes continuaram em execução mesmo após desconexão! <<<")
        return 1

if __name__ == "__main__":
    sys.exit(main())
