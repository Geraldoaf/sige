import os
import subprocess

print("=== [TEST 25] Application Files Unreachable from Sandbox ===")
print("Action: Trying to read the SIGE binaries, config and secrets...")

# Equivalente ao fix v1.4.0 do Judge0 ("move application location to prevent
# untrusted code from reading it") e ao v1.2.2 (arquivo de configuração
# legível por qualquer um). Os binários do SIGE ficam em /opt/sige, que não é
# montado no sandbox — diferente de /usr, que precisa ser montado por causa
# do interpretador e das bibliotecas.
alvos = [
    "/opt/sige/sige",
    "/opt/sige/sige-launch",
    "/usr/local/bin/sige",
    "/usr/local/bin/sige-launch",
    "/workspace/config.json",
    "/run/secrets/sige_api_key",
    "/etc/shadow",
]

for alvo in alvos:
    try:
        with open(alvo, "rb") as f:
            f.read(16)
        print(f"Result: UNEXPECTED - {alvo} e LEGIVEL pelo codigo do usuario!")
    except OSError as e:
        print(f"OK - {alvo} inacessivel ({type(e).__name__})")

# Mesmo que o binário privilegiado fosse alcançável, NoNewPrivs=1 impede
# ganhar as file capabilities e o seccomp mata as syscalls necessárias.
print("Action: Trying to execute the privileged launcher...")
for launcher in ("/opt/sige/sige-launch", "/usr/local/bin/sige-launch"):
    try:
        r = subprocess.run([launcher, "--help"], capture_output=True, timeout=10)
        print(f"Result: {launcher} executou com rc={r.returncode}"
              " (negativo = morto por sinal, esperado)")
    except OSError as e:
        print(f"OK - {launcher} nao executavel ({type(e).__name__})")

print("=== END TEST 25 ===")
