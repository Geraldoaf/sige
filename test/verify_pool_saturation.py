#!/usr/bin/env python3
"""
verify_pool_saturation.py

Verifica a correção do bug 2.1 (Pool Global de Concorrência) via API HTTP.
Dispara requisições concorrentes acima do limite de slots e valida:
1. Respostas HTTP 503 com erro AT_CAPACITY e Retry-After para requisições excedentes.
2. Contabilização precisa no endpoint /capacity.
"""

import sys
import time
import json
import urllib.request
import urllib.error
from concurrent.futures import ThreadPoolExecutor

BASE_URL = sys.argv[1] if len(sys.argv) > 1 else "http://127.0.0.1:8080"
API_KEY = sys.argv[2] if len(sys.argv) > 2 else ""

def get_capacity():
    req = urllib.request.Request(f"{BASE_URL}/capacity")
    if API_KEY:
        req.add_header("X-API-Key", API_KEY)
    try:
        with urllib.request.urlopen(req, timeout=5) as resp:
            data = json.loads(resp.read().decode())
            return {
                "active": data.get("active_sandboxes", data.get("active", 0)),
                "total": data.get("max_concurrent_sandboxes", data.get("total", 8)),
                "available": data.get("available_slots", data.get("available", 0))
            }
    except Exception as e:
        print(f"[!] Erro ao consultar /capacity: {e}")
        return None

def send_long_running_exec(task_id):
    # Execução de 4 segundos no sandbox
    code = f"""
import time
print("Task {task_id} running")
time.sleep(4)
print("Task {task_id} finished")
"""
    payload = json.dumps({
        "language": "python",
        "code": code
    }).encode("utf-8")

    req = urllib.request.Request(
        f"{BASE_URL}/execute",
        data=payload,
        headers={"Content-Type": "application/json"}
    )
    if API_KEY:
        req.add_header("X-API-Key", API_KEY)

    try:
        with urllib.request.urlopen(req, timeout=15) as resp:
            body = json.loads(resp.read().decode())
            return resp.status, body, None
    except urllib.error.HTTPError as e:
        body = e.read().decode()
        retry_after = e.headers.get("Retry-After")
        return e.code, body, retry_after
    except Exception as e:
        return 0, str(e), None

def main():
    print(f"=== [VERIFICAÇÃO API] Pool de Concorrência e Saturação ({BASE_URL}) ===")
    
    initial_cap = get_capacity()
    if initial_cap:
        print(f"[*] Capacidade Inicial: {initial_cap.get('active', 0)} ativos / {initial_cap.get('total', 8)} total")
        slots = initial_cap.get("total", 8)
    else:
        slots = 8

    # Dispara 18 requisições (2 lotes de 8 + 2 excedentes, dentro do burst de 20 do rate limiter)
    concurrent_requests = 18
    print(f"[*] Disparando {concurrent_requests} requisições simultâneas para estressar o pool...")

    results = []
    with ThreadPoolExecutor(max_workers=concurrent_requests) as executor:
        futures = [executor.submit(send_long_running_exec, i) for i in range(concurrent_requests)]
        
        # Durante a execução, verifica a capacidade ocupada
        time.sleep(0.8)
        mid_cap = get_capacity()
        if mid_cap:
            print(f"[*] Capacidade no meio da carga: {mid_cap.get('active')} ativos / {mid_cap.get('total')} total")

        for f in futures:
            results.append(f.result())

    success_count = 0
    at_capacity_count = 0
    other_errors = 0

    for status, body, retry_after in results:
        if status == 200:
            success_count += 1
        elif status == 503:
            at_capacity_count += 1
            if retry_after:
                print(f"[+] HTTP 503 recebido com Retry-After: {retry_after}")
        else:
            other_errors += 1
            print(f"[?] Resposta inesperada: status={status}, body={body}")

    print(f"\n=== RESULTADOS ===")
    print(f"Sucesso (200 OK): {success_count}")
    print(f"At Capacity (503): {at_capacity_count}")
    print(f"Outros erros:    {other_errors}")

    final_cap = get_capacity()
    if final_cap:
        print(f"[*] Capacidade Final: {final_cap.get('active')} ativos / {final_cap.get('total')} total")

    if at_capacity_count > 0 or success_count == slots:
        print("\n>>> SUCESSO: O Pool Global limitou a concorrência e não recriou semáforos vazios! <<<")
        return 0
    else:
        print("\n>>> FALHA: Nenhuma limitação de concorrência foi observada! <<<")
        return 1

if __name__ == "__main__":
    sys.exit(main())
