package sandbox

import (
	"context"
	"errors"
	"sige/internal/engine"
)

// ErrAtCapacity indica que o servidor já está com o número máximo de
// sandboxes em execução e a espera por uma vaga esgotou.
var ErrAtCapacity = errors.New("servidor na capacidade máxima de sandboxes simultâneos")

// acquireSlot reserva uma vaga global de execução utilizando o engine.GlobalPool().
func acquireSlot() (func(), error) {
	return acquireSlotContext(context.Background())
}

// acquireSlotContext reserva uma vaga global respeitando o cancelamento do context informado.
func acquireSlotContext(ctx context.Context) (func(), error) {
	if ctx == nil {
		ctx = context.Background()
	}
	release, err := engine.GlobalPool().Acquire(ctx)
	if err != nil {
		if errors.Is(err, engine.ErrPoolSaturated) {
			return nil, ErrAtCapacity
		}
		return nil, err
	}
	return release, nil
}
