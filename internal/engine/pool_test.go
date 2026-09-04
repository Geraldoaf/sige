package engine_test

import (
	"context"
	"sige/internal/engine"
	"sync"
	"testing"
	"time"
)

func TestCapacityPool_AcquireAndRelease(t *testing.T) {
	pool := engine.NewCapacityPool(2, 50*time.Millisecond)

	active, total := pool.Stats()
	if active != 0 || total != 2 {
		t.Fatalf("esperado 0/2 ativo, obtido %d/%d", active, total)
	}

	ctx := context.Background()

	// 1º slot
	release1, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("falha ao adquirir slot 1: %v", err)
	}

	// 2º slot
	release2, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("falha ao adquirir slot 2: %v", err)
	}

	active, _ = pool.Stats()
	if active != 2 {
		t.Errorf("esperado 2 ativos, obtido %d", active)
	}

	// 3º slot deve estourar o timeout e retornar ErrPoolSaturated
	_, err = pool.Acquire(ctx)
	if err != engine.ErrPoolSaturated {
		t.Errorf("esperado ErrPoolSaturated, obtido %v", err)
	}

	// Libera um slot
	release1()

	active, _ = pool.Stats()
	if active != 1 {
		t.Errorf("esperado 1 ativo apos release, obtido %d", active)
	}

	// Agora o 3º slot deve conseguir alocar
	release3, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("esperado conseguir adquirir slot liberado: %v", err)
	}

	release2()
	release3()

	active, _ = pool.Stats()
	if active != 0 {
		t.Errorf("esperado 0 ativos apos todas as liberacoes, obtido %d", active)
	}
}

func TestCapacityPool_ConcurrentAccess(t *testing.T) {
	pool := engine.NewCapacityPool(5, 500*time.Millisecond)
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			release, err := pool.Acquire(context.Background())
			if err == nil {
				time.Sleep(10 * time.Millisecond)
				release()
			}
		}()
	}

	wg.Wait()
	active, _ := pool.Stats()
	if active != 0 {
		t.Errorf("esperado 0 ativos ao final do teste concorrente, obtido %d", active)
	}
}
