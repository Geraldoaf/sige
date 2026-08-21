echo "=== [TEST 13] Orphan Process Reaping (PID namespace death) ==="
echo "Action: Spawning a background 'sleep 300' and exiting immediately..."

sleep 300 &
BGPID=$!
echo "Spawned background sleep with PID $BGPID, exiting now without waiting."
echo "=== END TEST 13 ==="

# Não dá pra verificar isso só pelo stdout: o que importa é observar de
# FORA do sandbox. Espera-se que:
#   1) esta execução termine quase instantaneamente (Duration ~ms, não 300s)
#      na resposta da API, já que o bash (PID 1 do namespace de PID desta
#      execução, ver cmd/internal_launch.go) sai sem esperar o filho;
#   2) o kernel mata TODO o namespace de PID quando o PID 1 dele termina —
#      então, rodando `ps aux | grep "sleep 300"` no container logo depois
#      dessa chamada à API, não deve sobrar nenhum processo "sleep" órfão.
exit 0
