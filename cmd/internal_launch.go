package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"sige/internal/sandbox"

	"github.com/spf13/cobra"
)

var (
	syncFd            int
	rootfs            string
	workspace         string
	tmpLimitMB        int
	fileLimitMB       int
	nofileLimit       int
	workspaceWritable bool
)

// internalLaunchCmd sincroniza com o cgroup do daemon e inicia o processo filho nos novos namespaces.
var internalLaunchCmd = &cobra.Command{
	Use:    "internal-launch",
	Hidden: true,
	Args:   cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if syncFd != -1 {
			f := os.NewFile(uintptr(syncFd), "sync-pipe")
			if f != nil {
				buf := make([]byte, 1)
				_, _ = f.Read(buf)
				f.Close()
			}
		}

		self, err := os.Executable()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao obter caminho do executável: %v\n", err)
			os.Exit(1)
		}

		// Cria processo filho nos novos namespaces (PID, UTS, NET, NS, IPC)
		childArgs := []string{
			"internal-launch-ns-child",
			"--rootfs", rootfs,
			"--workspace", workspace,
			"--tmp-limit", fmt.Sprintf("%d", tmpLimitMB),
			"--file-limit", fmt.Sprintf("%d", fileLimitMB),
			"--nofile-limit", fmt.Sprintf("%d", nofileLimit),
		}
		if workspaceWritable {
			childArgs = append(childArgs, "--workspace-writable")
		}
		childArgs = append(childArgs, "--")
		childArgs = append(childArgs, args...)

		child := exec.Command(self, childArgs...)
		child.Stdin = os.Stdin
		child.Stdout = os.Stdout
		child.Stderr = os.Stderr
		child.SysProcAttr = &syscall.SysProcAttr{
			Cloneflags: syscall.CLONE_NEWPID | syscall.CLONE_NEWUTS | syscall.CLONE_NEWNET |
				syscall.CLONE_NEWNS | syscall.CLONE_NEWIPC,
		}

		if err := child.Run(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				os.Exit(exitErr.ExitCode())
			}
			fmt.Fprintf(os.Stderr, "Erro ao executar processo filho do namespace de PID: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	},
}

// internalLaunchNSChildCmd isola o sistema de arquivos (pivot_root), aplica seccomp e executa o comando do usuário.
var internalLaunchNSChildCmd = &cobra.Command{
	Use:    "internal-launch-ns-child",
	Hidden: true,
	Args:   cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		if rootfs != "" && workspace != "" {
			if err := sandbox.ConfigureSandboxNamespace(rootfs, workspace, tmpLimitMB, fileLimitMB, nofileLimit, workspaceWritable); err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao configurar namespaces do sandbox: %v\n", err)
				os.Exit(1)
			}
		}

		targetCmd := args[0]
		targetArgs := args[1:]

		path, err := exec.LookPath(targetCmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Comando não encontrado no sandbox: %v\n", err)
			os.Exit(1)
		}

		if err := sandbox.ApplySeccompFilter(); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao aplicar filtro Seccomp: %v\n", err)
			os.Exit(1)
		}

		cleanEnv := []string{
			"PATH=/usr/local/bin:/usr/bin:/bin",
			"HOME=/tmp",
			"LANG=C.UTF-8",
			"TERM=xterm-256color",
		}
		err = syscall.Exec(path, append([]string{targetCmd}, targetArgs...), cleanEnv)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao executar syscall.Exec: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	internalLaunchCmd.Flags().IntVar(&syncFd, "sync-fd", -1, "File descriptor para sincronização")
	internalLaunchCmd.Flags().StringVar(&rootfs, "rootfs", "", "Caminho do rootfs temporário para o sandbox")
	internalLaunchCmd.Flags().StringVar(&workspace, "workspace", "", "Diretório de trabalho a ser montado no sandbox")
	internalLaunchCmd.Flags().IntVar(&tmpLimitMB, "tmp-limit", 64, "Limite de tamanho para /tmp em MB")
	internalLaunchCmd.Flags().IntVar(&fileLimitMB, "file-limit", 15, "Limite de tamanho máximo para arquivos gerados em MB")
	internalLaunchCmd.Flags().IntVar(&nofileLimit, "nofile-limit", 256, "Limite de quantidade máxima de arquivos/soquetes abertos")
	internalLaunchCmd.Flags().BoolVar(&workspaceWritable, "workspace-writable", false, "Monta /workspace como leitura-escrita (só para compilação)")
	rootCmd.AddCommand(internalLaunchCmd)

	internalLaunchNSChildCmd.Flags().StringVar(&rootfs, "rootfs", "", "Caminho do rootfs temporário para o sandbox")
	internalLaunchNSChildCmd.Flags().StringVar(&workspace, "workspace", "", "Diretório de trabalho a ser montado no sandbox")
	internalLaunchNSChildCmd.Flags().IntVar(&tmpLimitMB, "tmp-limit", 64, "Limite de tamanho para /tmp em MB")
	internalLaunchNSChildCmd.Flags().IntVar(&fileLimitMB, "file-limit", 15, "Limite de tamanho máximo para arquivos gerados em MB")
	internalLaunchNSChildCmd.Flags().IntVar(&nofileLimit, "nofile-limit", 256, "Limite de quantidade máxima de arquivos/soquetes abertos")
	internalLaunchNSChildCmd.Flags().BoolVar(&workspaceWritable, "workspace-writable", false, "Monta /workspace como leitura-escrita (só para compilação)")
	rootCmd.AddCommand(internalLaunchNSChildCmd)
}
