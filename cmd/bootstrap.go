package cmd

import (
	"fmt"
	"os"
	"sige/internal/cgroups"
	"sige/internal/sandbox"
	"syscall"
)

// bootstrapPrivileged prepara o ambiente privilegiado (seccomp base e delegação de cgroups)
// e realiza o rebaixamento seguro de privilégios para o usuário UID/GID 1001.
func bootstrapPrivileged() {
	if os.Getuid() == 0 {
		// Aplica filtro seccomp base
		if err := sandbox.ApplyBaselineSeccompFilter(); err != nil {
			fmt.Fprintf(os.Stderr, "[sige] Erro ao instalar filtro Seccomp base: %v\n", err)
			os.Exit(1)
		}

		// Remonta /sys/fs/cgroup em modo leitura-escrita
		if err := syscall.Mount("", "/sys/fs/cgroup", "", syscall.MS_REMOUNT, ""); err != nil {
			fmt.Fprintf(os.Stderr, "[sige] Aviso: não foi possível remontar /sys/fs/cgroup como rw: %v\n", err)
		}

		// Configura delegação de controladores cgroups v2 (+cpu +memory +pids)
		cgroups.SetupDelegation()

		// Cria e delega subdiretório de cgroup para o usuário sige (1001:1001)
		delegatedPaths := []string{
			"/sys/fs/cgroup/sige",
			"/sys/fs/cgroup/sige/cgroup.procs",
			"/sys/fs/cgroup/sige/cgroup.subtree_control",
			"/sys/fs/cgroup/sige/cgroup.threads",
		}
		if err := os.MkdirAll(delegatedPaths[0], 0755); err != nil {
			fmt.Fprintf(os.Stderr, "[sige] Erro ao criar diretório cgroup: %v\n", err)
			os.Exit(1)
		}
		for _, p := range delegatedPaths {
			if err := os.Chown(p, 1001, 1001); err != nil {
				fmt.Fprintf(os.Stderr, "[sige] Aviso: não foi possível delegar %s: %v\n", p, err)
			}
		}

		// Move o processo do daemon para o cgroup /sige/daemon
		if err := os.MkdirAll("/sys/fs/cgroup/sige/daemon", 0755); err != nil {
			fmt.Fprintf(os.Stderr, "[sige] Aviso: não foi possível criar /sige/daemon: %v\n", err)
		}
		if err := cgroups.AddProcessToCgroup("/sige/daemon", os.Getpid()); err != nil {
			fmt.Fprintf(os.Stderr, "[sige] Aviso: não foi possível mover o daemon para /sige/daemon: %v\n", err)
		}

		// Rebaixa GID e UID para 1001
		if err := syscall.Setgid(1001); err != nil {
			fmt.Fprintf(os.Stderr, "[sige] Erro ao dropar GID: %v\n", err)
			os.Exit(1)
		}
		if err := syscall.Setuid(1001); err != nil {
			fmt.Fprintf(os.Stderr, "[sige] Erro ao dropar UID: %v\n", err)
			os.Exit(1)
		}

		// Reexecuta o processo para aplicar as credenciais a todas as threads
		self, err := os.Executable()
		if err != nil {
			fmt.Fprintf(os.Stderr, "[sige] Erro ao obter caminho do executável: %v\n", err)
			os.Exit(1)
		}
		if err := syscall.Exec(self, os.Args, os.Environ()); err != nil {
			fmt.Fprintf(os.Stderr, "[sige] Erro ao re-executar sem privilégios: %v\n", err)
			os.Exit(1)
		}
	}
}
