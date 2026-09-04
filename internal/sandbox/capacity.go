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
	release, err := engine.GlobalPool().Acquire(context.Background())
	if err != nil {
		if errors.Is(err, engine.ErrPoolSaturated) {
			return nil, ErrAtCapacity
		}
		return nil, err
	}
	return release, nil
}
