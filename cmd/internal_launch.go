package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
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
			fmt.Fprintf(os.Stderr, "Error obtaining executable path: %v\n", err)
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
				if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok {
					if ws.Signaled() {
						sig := ws.Signal()
						signal.Reset(sig)
						_ = syscall.Kill(syscall.Getpid(), sig)
						os.Exit(128 + int(sig))
					}
				}
				os.Exit(exitErr.ExitCode())
			}
			fmt.Fprintf(os.Stderr, "Error executing PID namespace child process: %v\n", err)
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
		// Trava a goroutine na thread do SO para que alterações de capabilities,
		// bounding set e credenciais não se percam em trocas de thread do runtime Go.
		runtime.LockOSThread()

		// Fail-closed: antes, argumentos vazios faziam este bloco ser PULADO e
		// o comando do usuário rodava mesmo assim — sem pivot_root, sem
		// rlimits, sem esvaziar o bounding set e sem rebaixar para nobody.
		// Isolamento não pode ser opcional: sem os caminhos, aborta.
		if rootfs == "" || workspace == "" {
			fmt.Fprintln(os.Stderr, "Error: --rootfs and --workspace are required; refusing to execute without isolation.")
			os.Exit(1)
		}

		if err := sandbox.ConfigureSandboxNamespace(rootfs, workspace, tmpLimitMB, fileLimitMB, nofileLimit, workspaceWritable); err != nil {
			fmt.Fprintf(os.Stderr, "Error configuring sandbox namespaces: %v\n", err)
			os.Exit(1)
		}

		targetCmd := args[0]
		targetArgs := args[1:]

		path, err := exec.LookPath(targetCmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Command not found in sandbox: %v\n", err)
			os.Exit(1)
		}

		if err := sandbox.ApplySeccompFilter(); err != nil {
			fmt.Fprintf(os.Stderr, "Error applying seccomp filter: %v\n", err)
			os.Exit(1)
		}

		cleanEnv := []string{
			"PATH=/usr/local/bin:/usr/bin:/bin",
			"HOME=/tmp",
			"LANG=C.UTF-8",
			"TERM=xterm-256color",
		}

		// Inicia o processo do usuário como processo filho (PID > 1 no namespace).
		// O processo atual (PID 1 do namespace) atua como mini-init para colher zumbis
		// e repassar sinais ao processo do usuário, garantindo semântica POSIX padrão.
		userCmd := exec.Command(path, targetArgs...)
		userCmd.Stdin = os.Stdin
		userCmd.Stdout = os.Stdout
		userCmd.Stderr = os.Stderr
		userCmd.Env = cleanEnv

		sigChan := make(chan os.Signal, 8)
		signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)

		if err := userCmd.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting user process: %v\n", err)
			os.Exit(1)
		}

		go func() {
			for sig := range sigChan {
				if userCmd.Process != nil {
					_ = userCmd.Process.Signal(sig)
				}
			}
		}()

		waitErr := userCmd.Wait()

		// Como PID 1 do namespace, colhe quaisquer outros processos filhos órfãos
		for {
			var ws syscall.WaitStatus
			p, _ := syscall.Wait4(-1, &ws, syscall.WNOHANG, nil)
			if p <= 0 {
				break
			}
		}

		if waitErr != nil {
			if exitErr, ok := waitErr.(*exec.ExitError); ok {
				if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok {
					if ws.Signaled() {
						sig := ws.Signal()
						signal.Reset(sig)
						_ = syscall.Kill(syscall.Getpid(), sig)
						os.Exit(128 + int(sig))
					}
				}
				os.Exit(exitErr.ExitCode())
			}
			os.Exit(1)
		}
		os.Exit(0)
	},
}

func init() {
	internalLaunchCmd.Flags().IntVar(&syncFd, "sync-fd", -1, "File descriptor for synchronization")
	internalLaunchCmd.Flags().StringVar(&rootfs, "rootfs", "", "Temporary rootfs path for the sandbox")
	internalLaunchCmd.Flags().StringVar(&workspace, "workspace", "", "Working directory to mount in the sandbox")
	internalLaunchCmd.Flags().IntVar(&tmpLimitMB, "tmp-limit", 64, "Size limit for /tmp in MB")
	internalLaunchCmd.Flags().IntVar(&fileLimitMB, "file-limit", 15, "Maximum file size limit in MB")
	internalLaunchCmd.Flags().IntVar(&nofileLimit, "nofile-limit", 256, "Maximum open files/sockets limit")
	internalLaunchCmd.Flags().BoolVar(&workspaceWritable, "workspace-writable", false, "Mount /workspace as read-write (compilation only)")
	rootCmd.AddCommand(internalLaunchCmd)

	internalLaunchNSChildCmd.Flags().StringVar(&rootfs, "rootfs", "", "Temporary rootfs path for the sandbox")
	internalLaunchNSChildCmd.Flags().StringVar(&workspace, "workspace", "", "Working directory to mount in the sandbox")
	internalLaunchNSChildCmd.Flags().IntVar(&tmpLimitMB, "tmp-limit", 64, "Size limit for /tmp in MB")
	internalLaunchNSChildCmd.Flags().IntVar(&fileLimitMB, "file-limit", 15, "Maximum file size limit in MB")
	internalLaunchNSChildCmd.Flags().IntVar(&nofileLimit, "nofile-limit", 256, "Maximum open files/sockets limit")
	internalLaunchNSChildCmd.Flags().BoolVar(&workspaceWritable, "workspace-writable", false, "Mount /workspace as read-write (compilation only)")
	rootCmd.AddCommand(internalLaunchNSChildCmd)
}
