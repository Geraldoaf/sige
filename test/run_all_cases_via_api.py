#!/usr/bin/env python3
"""
run_all_cases_via_api.py

Executa todos os 44 casos de teste reais de test/real_cases/ via API HTTP
contra o container Docker em execução (http://127.0.0.1:8080).
"""

import os
import sys
import json
import urllib.request
import urllib.error

BASE_URL = sys.argv[1] if len(sys.argv) > 1 else "http://127.0.0.1:8080"
API_KEY = sys.argv[2] if len(sys.argv) > 2 else ""

def execute_payload(lang, code):
    payload = json.dumps({
        "language": lang,
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
        with urllib.request.urlopen(req, timeout=20) as resp:
            data = json.loads(resp.read().decode())
            return resp.status, data
    except urllib.error.HTTPError as e:
        body = e.read().decode()
        try:
            return e.code, json.loads(body)
        except Exception:
            return e.code, {"raw": body}
    except Exception as e:
        return 0, {"error": str(e)}

def main():
    cases_dir = os.path.join(os.path.dirname(__file__), "real_cases")
    files = sorted([f for f in os.listdir(cases_dir) if f.endswith((".py", ".c", ".cpp", ".sh"))])

    print(f"=== Executando {len(files)} casos de teste reais na API ({BASE_URL}) ===")
    
    passed_or_expected = 0
    total = len(files)

    for f in files:
        filepath = os.path.join(cases_dir, f)
        with open(filepath, "r", encoding="utf-8", errors="ignore") as fp:
            content = fp.read()

        if f.endswith(".py"):
            lang = "python"
        elif f.endswith(".c"):
            lang = "c"
        elif f.endswith(".cpp"):
            lang = "cpp"
        elif f.endswith(".sh"):
            lang = "bash"
        else:
            continue

        status_code, resp = execute_payload(lang, content)
        
        exec_status = resp.get("execution", {}).get("status", resp.get("error", {}).get("code", "unknown"))
        result = resp.get("result", "error")
        exit_code = resp.get("execution", {}).get("exit_code", -1)

        # Asserções específicas de segurança para os casos corrigidos
        alert = ""
        if f == "42_forged_stderr_detection.c":
            if exec_status == "file_size_exceeded":
                alert = " [CRÍTICO: Vulnerabilidade! Classificou como file_size_exceeded]"
            else:
                alert = " [CORRETO: Rejeitou forjamento de stderr]"

        elif f == "40_socket_af_vsock_alg_blocked.c":
            stdout = resp.get("execution", {}).get("stdout", "")
            if "FAIL:" in stdout:
                alert = " [FALHA: Soquete proibido permitido]"
            else:
                alert = " [CORRETO: Soquetes proibidos bloqueados]"

        elif f == "45_not_pid_one.c":
            stdout = resp.get("execution", {}).get("stdout", "")
            if "PASS:" in stdout and exit_code == 0:
                alert = " [CORRETO: Mini-init ativo, usuário PID > 1]"
            else:
                alert = " [FALHA: Usuário rodando como PID 1]"

        print(f"[{f:40}] HTTP {status_code} | Result: {result:9} | Status: {exec_status:15} | Exit: {exit_code}{alert}")
        passed_or_expected += 1

    print(f"\nTotal de casos executados: {passed_or_expected}/{total}")
    return 0

if __name__ == "__main__":
    sys.exit(main())
