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
	syncFd      int
	rootfs      string
	workspace   string
	tmpLimitMB  int
	fileLimitMB int
	nofileLimit int
)

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

		if rootfs != "" && workspace != "" {
			if err := sandbox.ConfigureSandboxNamespace(rootfs, workspace, tmpLimitMB, fileLimitMB, nofileLimit); err != nil {
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
	rootCmd.AddCommand(internalLaunchCmd)
}
