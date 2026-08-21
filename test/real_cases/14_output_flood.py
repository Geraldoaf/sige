print("=== [TEST 14] Stdout Flood (1MB output cap) ===")
print("Action: Printing indefinitely to exceed the output limit...")

while True:
    print("A" * 65536)

# Nunca chega aqui. Esperado: status "output_limit_exceeded" na resposta da
# API (ver limitedBuffer/onOutputLimitExceeded em internal/sandbox/executor.go),
# stdout truncado perto de constants.DefaultMaxOutputBytes (1MB), e a
# execução INTEIRA encerrada (não só o print parado) — confirma que o
# estouro mata o processo via cgroup.kill, igual a um timeout, em vez de só
# descartar a saída excedente e deixar o loop infinito rodando.
