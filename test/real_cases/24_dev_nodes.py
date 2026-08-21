import os
import subprocess

print("=== [TEST 24] Minimal /dev Availability ===")
print("Action: Exercising the device files legitimate programs expect...")

# Antes de mountMinimalDev (ver internal/sandbox/namespace.go) o sandbox não
# tinha /dev nenhum, e tudo abaixo falhava com FileNotFoundError — não por
# política de segurança, mas por ausência do diretório. Isso quebrava código
# perfeitamente comum.
print("dev existe:", os.path.isdir("/dev"))
print("conteudo:", sorted(os.listdir("/dev")))

with open("/dev/null", "w") as f:
    f.write("descartado")
print("OK - escrita em /dev/null")

print("OK - leitura de /dev/zero:", open("/dev/zero", "rb").read(4).hex())
print("OK - leitura de /dev/urandom:", open("/dev/urandom", "rb").read(4).hex())

subprocess.run(["echo", "silenciado"], stdout=subprocess.DEVNULL)
print("OK - subprocess.DEVNULL")

# Symlinks padrão apontando para /proc.
print("OK - /dev/stdout ->", os.readlink("/dev/stdout"))
print("OK - /dev/fd ->", os.readlink("/dev/fd"))

# /dev deve ser somente-leitura: os device files existem, mas o código do
# usuário não pode criar nada novo ali.
try:
    open("/dev/arquivo_novo", "w").close()
    print("Result: UNEXPECTED - created a file in /dev, it is NOT read-only!")
except OSError as e:
    print(f"OK - /dev read-only: {e}")

# Dispositivos deliberadamente ausentes (o sandbox não tem terminal).
for ausente in ("/dev/tty", "/dev/console", "/dev/ptmx"):
    print(f"OK - {ausente} ausente por design:", not os.path.exists(ausente))

print("=== END TEST 24 ===")
