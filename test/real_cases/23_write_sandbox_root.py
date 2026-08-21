print("=== [TEST 23] Sandbox Root (/) Read-Only ===")
print("Action: Attempting to write files directly into the sandbox root '/'...")

# "/" é o rootfs temporário desta execução, que fica no disco do CONTAINER
# (não em tmpfs). Antes do remount read-only em ConfigureSandboxNamespace,
# este diretório era gravável — e como RLIMIT_FSIZE limita o tamanho de cada
# arquivo mas não a QUANTIDADE, um loop como o abaixo consumia disco real do
# container até o fim da execução.
try:
    with open("/escrita_na_raiz.txt", "w") as f:
        f.write("nao deveria conseguir escrever aqui")
    print("Result: UNEXPECTED - wrote to '/' , sandbox root is NOT read-only!")
except OSError as e:
    print(f"Result: BLOCKED - {e}")

# Criar diretório na raiz também deve falhar.
import os

try:
    os.mkdir("/diretorio_na_raiz")
    print("Result: UNEXPECTED - created a directory in '/'!")
except OSError as e:
    print(f"Result: BLOCKED (mkdir) - {e}")

# Controle: /tmp continua gravável (é tmpfs próprio, com limite de tamanho),
# senão o sandbox ficaria inutilizável para programas legítimos.
try:
    with open("/tmp/controle.txt", "w") as f:
        f.write("ok")
    print("Control: /tmp still writable, as expected")
except OSError as e:
    print(f"Control: UNEXPECTED - /tmp should be writable but failed: {e}")

print("=== END TEST 23 ===")
