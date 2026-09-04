package engine

import (
	"context"
	"errors"
	"os"
	"strconv"
	"sync/atomic"
	"time"
)

var (
	// ErrPoolSaturated indica que todos os slots de sandbox estão ocupados e o timeout esgotou.
	ErrPoolSaturated = errors.New("capacidade máxima de sandboxes atingida: timeout de espera na fila esgotado")
)

// CapacityPool gerencia a concorrência global de sandboxes no servidor.
type CapacityPool struct {
	sem        chan struct{}
	maxSlots   int
	activeRuns int64
	queueWait  time.Duration
}

// NewCapacityPool cria uma instância com o teto global especificado.
func NewCapacityPool(maxSlots int, queueWait time.Duration) *CapacityPool {
	if maxSlots <= 0 {
		maxSlots = 8
	}
	if queueWait <= 0 {
		queueWait = 5 * time.Second
	}
	return &CapacityPool{
		sem:       make(chan struct{}, maxSlots),
		maxSlots:  maxSlots,
		queueWait: queueWait,
	}
}

// GlobalPool inicializa o pool lendo SIGE_MAX_CONCURRENT_SANDBOXES.
func GlobalPool() *CapacityPool {
	slots := 8
	if v := os.Getenv("SIGE_MAX_CONCURRENT_SANDBOXES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			slots = n
		}
	}
	return NewCapacityPool(slots, 5*time.Second)
}

// Acquire tenta obter um slot no pool. Se o timeout de espera expirar ou o ctx for cancelado, retorna ErrPoolSaturated.
func (p *CapacityPool) Acquire(ctx context.Context) (func(), error) {
	waitCtx, cancel := context.WithTimeout(ctx, p.queueWait)
	defer cancel()

	select {
	case p.sem <- struct{}{}:
		atomic.AddInt64(&p.activeRuns, 1)
		release := func() {
			atomic.AddInt64(&p.activeRuns, -1)
			<-p.sem
		}
		return release, nil
	case <-waitCtx.Done():
		return nil, ErrPoolSaturated
	}
}

// Stats retorna os slots ocupados no momento e o total de slots configurados.
func (p *CapacityPool) Stats() (active int64, total int) {
	return atomic.LoadInt64(&p.activeRuns), p.maxSlots
}

// MaxSlots retorna a capacidade total.
func (p *CapacityPool) MaxSlots() int {
	return p.maxSlots
}
