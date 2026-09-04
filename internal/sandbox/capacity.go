package sandbox

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ErrAtCapacity indica que o servidor já está com o número máximo de
// sandboxes em execução e a espera por uma vaga esgotou.
var ErrAtCapacity = errors.New("servidor na capacidade máxima de sandboxes simultâneos")

const (
	// defaultMaxConcurrent é o teto GLOBAL de sandboxes simultâneos no
	// processo inteiro. Não confundir com maxConcurrentSandboxes (em
	// internal/api/engines.go), que limita o paralelismo DENTRO de uma única
	// requisição multi_evaluation.
	//
	// Sem este teto global, o limite por requisição não protegia nada: N
	// clientes concorrentes produziam N × 4 sandboxes, e o rate limiter (por
	// IP) não impede N clientes distintos.
	//
	// Ao ajustar, respeite a relação com o limite de memória do container:
	//   mem_limit >= maxConcurrent × ceiling_memory_mb + folga do daemon
	// (ver docker-compose.yml).
	defaultMaxConcurrent = 8

	// maxWait é quanto uma execução espera por uma vaga antes de desistir.
	// Preferimos enfileirar brevemente a rejeitar de imediato — picos curtos
	// são comuns —, mas sem passar do WriteTimeout do servidor HTTP.
	maxWait = 90 * time.Second
)

var (
	slotsOnce sync.Once
	slots     chan struct{}
)

// maxConcurrentGlobal lê SIGE_MAX_CONCURRENT_SANDBOXES, caindo no padrão
// quando ausente ou inválida.
func maxConcurrentGlobal() int {
	v := strings.TrimSpace(os.Getenv("SIGE_MAX_CONCURRENT_SANDBOXES"))
	if v == "" {
		return defaultMaxConcurrent
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return defaultMaxConcurrent
	}
	return n
}

// acquireSlot reserva uma vaga global de execução, bloqueando até no máximo
// maxWait. Devolve a função de liberação.
func acquireSlot() (func(), error) {
	slotsOnce.Do(func() {
		slots = make(chan struct{}, maxConcurrentGlobal())
	})

	timer := time.NewTimer(maxWait)
	defer timer.Stop()

	select {
	case slots <- struct{}{}:
		var once sync.Once
		return func() { once.Do(func() { <-slots }) }, nil
	case <-timer.C:
		return nil, ErrAtCapacity
	}
}
