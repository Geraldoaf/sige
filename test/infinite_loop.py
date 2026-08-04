import time

print("Script de teste iniciado. Entrando em loop infinito...")
print("Pressione Ctrl+C se estiver rodando fora do sandbox.")

try:
    while True:
        time.sleep(1)
except KeyboardInterrupt:
    print("\nScript interrompido manualmente.")
