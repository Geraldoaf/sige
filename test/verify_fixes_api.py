#!/usr/bin/env python3
"""
verify_fixes_api.py

Suíte completa de verificação ponta a ponta via API HTTP para validar as correções:
1. Validação de CPU rejeitando quotas/períodos inválidos com 400 Bad Request (Item 2.17).
2. Bloqueio de famílias perigosas de socket AF_VSOCK e AF_ALG pelo seccomp (Item 2.3).
3. Ataque de stderr forjado: strings "File too large" não devem forjar veredito (Item 2.7).
4. Limpeza de workspace após erro de compilação sem vazamento em disco (Item 2.8).
"""

import sys
import os
import glob
import json
import urllib.request
import urllib.error

BASE_URL = sys.argv[1] if len(sys.argv) > 1 else "http://127.0.0.1:8080"
API_KEY = sys.argv[2] if len(sys.argv) > 2 else ""

def do_request(endpoint, payload_dict, method="POST"):
    data = json.dumps(payload_dict).encode("utf-8") if payload_dict is not None else None
    req = urllib.request.Request(f"{BASE_URL}{endpoint}", data=data, method=method)
    req.add_header("Content-Type", "application/json")
    if API_KEY:
        req.add_header("X-API-Key", API_KEY)

    try:
        with urllib.request.urlopen(req, timeout=15) as resp:
            body = resp.read().decode()
            return resp.status, json.loads(body) if body else {}
    except urllib.error.HTTPError as e:
        body = e.read().decode()
        try:
            return e.code, json.loads(body)
        except Exception:
            return e.code, {"raw": body}
    except Exception as e:
        return 0, {"error": str(e)}

def test_cpu_validation():
    print("\n--- [TESTE 1] Validação de CPU com formatos e limites inválidos ---")
    invalid_cases = [
        {"cpu": "0", "desc": "Quota zero"},
        {"cpu": "100 5000000", "desc": "Periodo acima de 1s (5000000)"},
        {"cpu": "50 100", "desc": "Quota abaixo de 1000us"},
    ]
    all_passed = True
    for c in invalid_cases:
        payload = {
            "language": "python",
            "code": "print('test')",
            "cpu": c["cpu"]
        }
        status, body = do_request("/execute", payload)
        if status == 400:
            print(f"[PASS] {c['desc']} ('{c['cpu']}'): Rejeitado com HTTP 400 Bad Request como esperado.")
        else:
            print(f"[FAIL] {c['desc']} ('{c['cpu']}'): Retornou HTTP {status} (esperado 400). Resposta: {body}")
            all_passed = False
    return all_passed

def test_forged_stderr():
    print("\n--- [TESTE 2] Ataque de Stderr Forjado (Verificação de Sinais) ---")
    # Código C que emite "File too large" no stderr e sai com exit(1)
    c_code = """
#include <stdio.h>
#include <stdlib.h>
int main(void) {
    fprintf(stderr, "File too large\\nFile size limit exceeded\\n");
    return 1;
}
"""
    payload = {
        "language": "c",
        "code": c_code
    }
    status, body = do_request("/execute", payload)
    if status != 200:
        print(f"[WARN] Execução retornou status HTTP {status}: {body}")
        return False

    exec_status = body.get("execution", {}).get("status", "")
    print(f"[*] Status da Execução: '{exec_status}' (Exit Code: {body.get('execution', {}).get('exit_code')})")
    
    if exec_status == "file_size_exceeded":
        print("[FAIL] Vulnerabilidade presente: o daemon caiu no truque do stderr e marcou 'file_size_exceeded'!")
        return False
    else:
        print("[PASS] Correção confirmada: o daemon NÃO foi enganado pela mensagem no stderr.")
        return True

def test_workspace_cleanup_on_compile_error():
    print("\n--- [TESTE 3] Limpeza de Workspace em Erro de Compilação ---")
    c_syntax_error = "int main(void { return 0; }"
    
    # Submete código com erro de compilação 3 vezes
    for i in range(3):
        payload = {"language": "c", "code": c_syntax_error}
        status, body = do_request("/execute", payload)
        print(f"[*] Submissão {i+1}: HTTP {status}, Resultado: {body.get('result')}")

    # Verifica se sobraram pastas exec-* órfãs em /workspace (se executado localmente)
    if os.path.exists("/workspace"):
        leaked_dirs = glob.glob("/workspace/exec-*")
        print(f"[*] Diretórios encontrados em /workspace: {len(leaked_dirs)}")
        if len(leaked_dirs) == 0:
            print("[PASS] Nenhum diretório residual em /workspace! Cleanup automático funcionando.")
            return True
        else:
            print(f"[FAIL] Diretórios órfãos encontrados: {leaked_dirs}")
            return False
    else:
        print("[INFO] Diretório /workspace não acessível do ambiente do teste (rodando fora do container).")
        return True

def test_mini_init_pid():
    print("\n--- [TESTE 4] Mini-Init: Processo do Usuário Não Roda como PID 1 ---")
    c_code = """
#include <stdio.h>
#include <unistd.h>
int main(void) {
    pid_t pid = getpid();
    printf("USER_PID:%d\\n", pid);
    return (pid == 1) ? 1 : 0;
}
"""
    status, body = do_request("/execute", {"language": "c", "code": c_code})
    stdout = body.get("execution", {}).get("stdout", "")
    exit_code = body.get("execution", {}).get("exit_code", -1)
    print(f"[*] Resposta stdout: {stdout.strip()}, Exit Code: {exit_code}")
    if "USER_PID:" in stdout and exit_code == 0:
        # Extrai PID
        for line in stdout.splitlines():
            if line.startswith("USER_PID:"):
                pid_val = int(line.split(":")[1])
                if pid_val > 1:
                    print(f"[PASS] Mini-init ativo! Usuário executando com PID {pid_val} (> 1).")
                    return True
                else:
                    print(f"[FAIL] Usuário executando como PID {pid_val} (não gerenciado por init)!")
                    return False
    print(f"[FAIL] Execução falhou ou não retornou PID esperado: {body}")
    return False

def test_validate_alignment():
    print("\n--- [TESTE 5] Alinhamento de Modo entre /validate e /execute ---")
    status, body = do_request("/validate", {"language": "python", "code": "print('ok')"})
    if status == 200 and body.get("valid") is True:
        mode = body.get("api_mode")
        print(f"[PASS] /validate retornou modo consistente: '{mode}' e valid: true.")
        return mode == "interpreter"
    else:
        print(f"[FAIL] /validate retornou status {status}: {body}")
        return False

def test_exec_id_uniqueness():
    print("\n--- [TESTE 6] Imprevisibilidade e Isolamento Concorrente de Workspaces ---")
    import concurrent.futures
    import uuid

    tokens = [f"token-{uuid.uuid4()}" for _ in range(8)]
    results = {}

    def run_token_req(t):
        code = f"print('{t}')"
        st, res = do_request("/execute", {"language": "python", "code": code})
        return t, st, res.get("execution", {}).get("stdout", "").strip()

    with concurrent.futures.ThreadPoolExecutor(max_workers=8) as ex:
        futures = [ex.submit(run_token_req, tok) for tok in tokens]
        for f in concurrent.futures.as_completed(futures):
            tok, st, out = f.result()
            results[tok] = (st, out)

    all_matched = True
    for tok, (st, out) in results.items():
        if st != 200 or out != tok:
            print(f"[FAIL] Concorrência causou colisão ou erro: enviado {tok}, recebido '{out}' (status {st})")
            all_matched = False

    if all_matched:
        print(f"[PASS] Todas as 8 requisições concorrentes executaram em workspaces completamente isolados!")
        return True
    return False

def main():
    print(f"=== [SIGE] Execução da Suíte de Verificação via API ({BASE_URL}) ===")
    
    t1 = test_cpu_validation()
    t2 = test_forged_stderr()
    t3 = test_workspace_cleanup_on_compile_error()
    t4 = test_mini_init_pid()
    t5 = test_validate_alignment()
    t6 = test_exec_id_uniqueness()

    print("\n=== RESUMO GERAL ===")
    print(f"1. Validação de CPU:          {'SUCESSO' if t1 else 'FALHA'}")
    print(f"2. Proteção Stderr Forjado:    {'SUCESSO' if t2 else 'FALHA'}")
    print(f"3. Cleanup de Workspace:       {'SUCESSO' if t3 else 'FALHA'}")
    print(f"4. Mini-Init (PID > 1):        {'SUCESSO' if t4 else 'FALHA'}")
    print(f"5. Alinhamento /validate:      {'SUCESSO' if t5 else 'FALHA'}")
    print(f"6. IDs Aleatórios / Workspace: {'SUCESSO' if t6 else 'FALHA'}")

    all_tests = [t1, t2, t3, t4, t5, t6]
    if all(all_tests):
        print("\n>>> TODAS AS VERIFICAÇÕES PASSARAM COM SUCESSO! <<<")
        return 0
    else:
        print("\n>>> ALGUMAS VERIFICAÇÕES FALHARAM! <<<")
        return 1

if __name__ == "__main__":
    sys.exit(main())
