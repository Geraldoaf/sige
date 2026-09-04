package sandbox

import (
	"fmt"
	"os"
	"regexp"

	"sige/internal/cgroups"
)

// Execute valida os parâmetros, cria o cgroup e executa o comando no sandbox.
func Execute(config Config, command string, args []string) (ExecutionResult, error) {
	matched, err := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, config.Name)
	if err != nil {
		return ExecutionResult{}, fmt.Errorf("error validating sandbox name: %w", err)
	}
	if !matched {
		return ExecutionResult{}, fmt.Errorf("invalid sandbox name '%s': must contain only alphanumeric characters, hyphens, and underscores", config.Name)
	}

	if !cgroups.VerifyCgroupsVersion() {
		return ExecutionResult{}, fmt.Errorf("system does not support cgroup v2 or is not in Unified Mode")
	}

	// Teto global de sandboxes simultâneos. Fica aqui, e não na camada HTTP,
	// porque este é o ponto único por onde passam TODOS os caminhos: os três
	// modos da API, a compilação C/C++ e o CLI run-task.
	release, err := acquireSlot()
	if err != nil {
		return ExecutionResult{Status: "at_capacity"}, err
	}
	defer release()

	memBytes := config.GetMemoryBytes()
	cpuFormatted := config.GetFormattedCPU()

	mgr, err := cgroups.CreateCgroup(config.Name, memBytes, cpuFormatted)
	if err != nil {
		return ExecutionResult{}, fmt.Errorf("error creating cgroup: %w", err)
	}

	// Garante o encerramento dos processos e limpeza do cgroup ao finalizar
	defer func() {
		if err := cgroups.KillAndDeleteCgroup(config.Name); err != nil {
			fmt.Fprintf(os.Stderr, "[SIGE] Aviso: falha ao limpar cgroup da execução: %v\n", err)
		}
	}()

	return Run(mgr, config, command, args...)
}
