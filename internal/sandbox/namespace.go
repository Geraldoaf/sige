package sandbox

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"sige/internal/constants"
)

// ConfigureSandboxNamespace prepara os pontos de montagem, pivot_root, limites de recursos e drop para usuário nobody.
func ConfigureSandboxNamespace(rootfs, workspace string, tmpLimitMB, fileLimitMB, nofileLimit int, workspaceWritable bool) error {
	if err := syscall.Mount("", "/", "", syscall.MS_REC|syscall.MS_PRIVATE, ""); err != nil {
		return fmt.Errorf("error configuring mount propagation as private: %w", err)
	}

	if err := syscall.Mount(rootfs, rootfs, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("error bind-mounting temporary rootfs: %w", err)
	}

	// Monta diretórios essenciais do sistema como somente-leitura
	const commonFlags = syscall.MS_BIND | syscall.MS_RDONLY | syscall.MS_NOSUID | syscall.MS_NODEV
	dirsToMount := map[string]uintptr{
		"/bin":   commonFlags,
		"/lib":   commonFlags,
		"/lib64": commonFlags,
		"/usr":   commonFlags,
		"/etc":   commonFlags | syscall.MS_NOEXEC,
	}
	for _, dir := range []string{"/bin", "/lib", "/lib64", "/usr", "/etc"} {
		if _, err := os.Stat(dir); err != nil {
			if os.IsNotExist(err) {
				continue
			}
		}

		dest := filepath.Join(rootfs, dir)
		if err := os.MkdirAll(dest, 0755); err != nil {
			return fmt.Errorf("error creating mount point %s: %w", dest, err)
		}

		if err := syscall.Mount(dir, dest, "", syscall.MS_BIND, ""); err != nil {
			return fmt.Errorf("error mounting %s to %s: %w", dir, dest, err)
		}

		if err := syscall.Mount("", dest, "", dirsToMount[dir]|syscall.MS_REMOUNT, ""); err != nil {
			return fmt.Errorf("error remounting %s with protective flags: %w", dest, err)
		}
	}

	// Monta /proc isolado
	procDest := filepath.Join(rootfs, "proc")
	if err := os.MkdirAll(procDest, 0755); err != nil {
		return fmt.Errorf("error creating proc directory in rootfs: %w", err)
	}
	// MS_NOSUID|MS_NODEV|MS_NOEXEC: as mesmas proteções dos demais mounts,
	// que aqui faltavam (as flags eram 0).
	if err := syscall.Mount("proc", procDest, "proc",
		syscall.MS_NOSUID|syscall.MS_NODEV|syscall.MS_NOEXEC, "hidepid=2"); err != nil {
		return fmt.Errorf("error mounting proc: %w", err)
	}

	if err := hardenProc(procDest); err != nil {
		return err
	}

	// Monta /tmp em tmpfs
	tmpDest := filepath.Join(rootfs, "tmp")
	if err := os.MkdirAll(tmpDest, 0755); err != nil {
		return fmt.Errorf("error creating tmp directory in rootfs: %w", err)
	}
	if tmpLimitMB <= 0 {
		if err := syscall.Mount("tmpfs", tmpDest, "tmpfs", syscall.MS_RDONLY|syscall.MS_NOEXEC|syscall.MS_NOSUID|syscall.MS_NODEV, "size=0"); err != nil {
			return fmt.Errorf("error mounting read-only tmpfs to /tmp: %w", err)
		}
	} else {
		options := fmt.Sprintf("size=%dm", tmpLimitMB)
		if err := syscall.Mount("tmpfs", tmpDest, "tmpfs", syscall.MS_NOEXEC|syscall.MS_NOSUID|syscall.MS_NODEV, options); err != nil {
			return fmt.Errorf("error mounting tmpfs with size %dMB to /tmp: %w", tmpLimitMB, err)
		}
	}

	if err := mountMinimalDev(rootfs, tmpLimitMB); err != nil {
		return err
	}

	// Monta /workspace
	workspaceDest := filepath.Join(rootfs, "workspace")
	if err := os.MkdirAll(workspaceDest, 0755); err != nil {
		return fmt.Errorf("error creating workspace folder in rootfs: %w", err)
	}
	if err := syscall.Mount(workspace, workspaceDest, "", syscall.MS_BIND, ""); err != nil {
		return fmt.Errorf("error mounting workspace to %s: %w", workspaceDest, err)
	}

	workspaceFlags := uintptr(syscall.MS_BIND | syscall.MS_REMOUNT | syscall.MS_NOSUID | syscall.MS_NODEV)
	if !workspaceWritable {
		workspaceFlags |= syscall.MS_RDONLY
	}
	if err := syscall.Mount("", workspaceDest, "", workspaceFlags, ""); err != nil {
		return fmt.Errorf("error remounting workspace with protective flags: %w", err)
	}

	// Troca a raiz do filesystem com pivot_root
	oldRoot := filepath.Join(rootfs, "old_root")
	if err := os.MkdirAll(oldRoot, constants.OldRootPerms); err != nil {
		return fmt.Errorf("error creating old_root folder in rootfs: %w", err)
	}

	if err := syscall.PivotRoot(rootfs, oldRoot); err != nil {
		return fmt.Errorf("error executing pivot_root: %w", err)
	}

	if err := syscall.Chdir("/"); err != nil {
		return fmt.Errorf("error changing directory to new root /: %w", err)
	}

	if err := syscall.Unmount("/old_root", syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("error unmounting /old_root: %w", err)
	}

	_ = os.Remove("/old_root")

	// Remonta a nova raiz como somente-leitura
	rootFlags := uintptr(syscall.MS_BIND | syscall.MS_REMOUNT | syscall.MS_RDONLY |
		syscall.MS_NOSUID | syscall.MS_NODEV)
	if err := syscall.Mount("", "/", "", rootFlags, ""); err != nil {
		return fmt.Errorf("error remounting sandbox root as read-only: %w", err)
	}

	if err := syscall.Chdir("/workspace"); err != nil {
		return fmt.Errorf("error changing directory to /workspace: %w", err)
	}

	if err := applyResourceLimits(fileLimitMB, nofileLimit); err != nil {
		return fmt.Errorf("error applying resource limits (rlimits): %w", err)
	}

	if err := dropCapabilityBoundingSet(); err != nil {
		return fmt.Errorf("error dropping capability bounding set: %w", err)
	}

	// Rebaixa privilégios para usuário e grupo nobody
	if err := syscall.Setgid(constants.DefaultGIDNobody); err != nil {
		return fmt.Errorf("error dropping GID privileges to nobody: %w", err)
	}
	if err := syscall.Setuid(constants.DefaultUIDNobody); err != nil {
		return fmt.Errorf("error dropping UID privileges to nobody: %w", err)
	}

	return nil
}

// procMaskedFiles são arquivos de /proc que expõem estado do kernel e não têm
// uso legítimo para o código do usuário. São cobertos por um bind de
// /dev/null — a leitura devolve vazio em vez de conteúdo do kernel.
//
// /proc/kcore é o caso mais grave: é a imagem da memória física do kernel.
// /proc/sysrq-trigger aciona funções de emergência (inclusive reiniciar a
// máquina). Os demais vazam layout de memória e temporização, úteis para
// contornar KASLR. É o mesmo conjunto que o Docker mascara por padrão.
var procMaskedFiles = []string{
	"kcore", "keys", "key-users", "sysrq-trigger",
	"timer_list", "timer_stats", "sched_debug", "latency_stats", "interrupts",
}

// procMaskedDirs são diretórios de /proc cobertos por um tmpfs vazio e
// somente-leitura.
var procMaskedDirs = []string{"acpi", "asound", "scsi"}

// procReadonlyPaths permanecem visíveis (há código que lê /proc/sys), mas
// remontados somente-leitura. O código do usuário não tem capability para
// escrever neles de qualquer forma; isto é defesa em profundidade.
var procReadonlyPaths = []string{"bus", "fs", "irq", "sys"}

// hardenProc mascara os caminhos sensíveis de /proc dentro do rootfs.
// Executado antes do pivot_root, quando o /dev do container ainda é visível
// para servir de fonte do bind.
func hardenProc(procDest string) error {
	for _, nome := range procMaskedFiles {
		alvo := filepath.Join(procDest, nome)
		if _, err := os.Stat(alvo); err != nil {
			// Não existe neste kernel (timer_stats, por exemplo, foi
			// removido); nada a mascarar.
			continue
		}
		if err := syscall.Mount("/dev/null", alvo, "", syscall.MS_BIND, ""); err != nil {
			return fmt.Errorf("error masking /proc/%s: %w", nome, err)
		}
	}

	for _, nome := range procMaskedDirs {
		alvo := filepath.Join(procDest, nome)
		if _, err := os.Stat(alvo); err != nil {
			continue
		}
		flags := uintptr(syscall.MS_RDONLY | syscall.MS_NOSUID | syscall.MS_NODEV | syscall.MS_NOEXEC)
		if err := syscall.Mount("tmpfs", alvo, "tmpfs", flags, "size=0,mode=0555"); err != nil {
			return fmt.Errorf("error masking /proc/%s: %w", nome, err)
		}
	}

	for _, nome := range procReadonlyPaths {
		alvo := filepath.Join(procDest, nome)
		if _, err := os.Stat(alvo); err != nil {
			continue
		}
		if err := syscall.Mount(alvo, alvo, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
			return fmt.Errorf("error binding /proc/%s: %w", nome, err)
		}
		flags := uintptr(syscall.MS_BIND | syscall.MS_REMOUNT | syscall.MS_RDONLY |
			syscall.MS_NOSUID | syscall.MS_NODEV | syscall.MS_NOEXEC)
		if err := syscall.Mount("", alvo, "", flags, ""); err != nil {
			return fmt.Errorf("error remounting /proc/%s as read-only: %w", nome, err)
		}
	}

	return nil
}

// devNodes são os arquivos de dispositivo expostos no /dev do sandbox.
var devNodes = []string{"null", "zero", "full", "random", "urandom"}

// devSymlinks são links simbólicos padrão para descritores de arquivos e streams.
var devSymlinks = map[string]string{
	"fd":     "/proc/self/fd",
	"stdin":  "/proc/self/fd/0",
	"stdout": "/proc/self/fd/1",
	"stderr": "/proc/self/fd/2",
}

// mountMinimalDev monta os nós essenciais de /dev e o diretório /dev/shm em tmpfs.
func mountMinimalDev(rootfs string, tmpLimitMB int) error {
	devDest := filepath.Join(rootfs, "dev")
	if err := os.MkdirAll(devDest, 0755); err != nil {
		return fmt.Errorf("error creating dev directory in rootfs: %w", err)
	}

	if err := syscall.Mount("tmpfs", devDest, "tmpfs", syscall.MS_NOSUID|syscall.MS_NOEXEC, "size=1m,mode=0755"); err != nil {
		return fmt.Errorf("error mounting tmpfs to /dev: %w", err)
	}

	for _, node := range devNodes {
		src := filepath.Join("/dev", node)
		if _, err := os.Stat(src); err != nil {
			continue
		}

		dst := filepath.Join(devDest, node)
		f, err := os.OpenFile(dst, os.O_CREATE|os.O_RDONLY, 0666)
		if err != nil {
			return fmt.Errorf("error creating mount point for /dev/%s: %w", node, err)
		}
		f.Close()

		if err := syscall.Mount(src, dst, "", syscall.MS_BIND, ""); err != nil {
			return fmt.Errorf("error bind-mounting /dev/%s: %w", node, err)
		}
	}

	for name, target := range devSymlinks {
		if err := os.Symlink(target, filepath.Join(devDest, name)); err != nil && !os.IsExist(err) {
			return fmt.Errorf("error creating /dev/%s symlink: %w", name, err)
		}
	}

	// Monta /dev/shm em tmpfs
	shmDest := filepath.Join(devDest, "shm")
	if err := os.MkdirAll(shmDest, 0777); err != nil {
		return fmt.Errorf("error creating /dev/shm directory: %w", err)
	}
	shmFlags := uintptr(syscall.MS_NOSUID | syscall.MS_NODEV | syscall.MS_NOEXEC)
	shmOptions := fmt.Sprintf("size=%dm,mode=1777", tmpLimitMB)
	if tmpLimitMB <= 0 {
		shmFlags |= syscall.MS_RDONLY
		shmOptions = "size=0,mode=1777"
	}
	if err := syscall.Mount("tmpfs", shmDest, "tmpfs", shmFlags, shmOptions); err != nil {
		return fmt.Errorf("error mounting tmpfs to /dev/shm: %w", err)
	}

	// Remonta /dev como somente-leitura
	if err := syscall.Mount("", devDest, "", syscall.MS_BIND|syscall.MS_REMOUNT|syscall.MS_RDONLY|syscall.MS_NOSUID|syscall.MS_NOEXEC, ""); err != nil {
		return fmt.Errorf("error remounting /dev as read-only: %w", err)
	}

	return nil
}

const prCapBsetDrop = 24

// dropCapabilityBoundingSet remove todas as capacidades do bounding set do processo.
func dropCapabilityBoundingSet() error {
	lastCap := 40
	if data, err := os.ReadFile("/proc/sys/kernel/cap_last_cap"); err == nil {
		if parsed, convErr := strconv.Atoi(strings.TrimSpace(string(data))); convErr == nil {
			lastCap = parsed
		}
	}

	for cap := 0; cap <= lastCap; cap++ {
		if _, _, errno := syscall.Syscall6(syscall.SYS_PRCTL, prCapBsetDrop, uintptr(cap), 0, 0, 0, 0); errno != 0 {
			if errno == syscall.EINVAL {
				continue
			}
			return fmt.Errorf("prctl(PR_CAPBSET_DROP, %d): %w", cap, errno)
		}
	}

	return nil
}

// applyResourceLimits aplica limites de tamanho de arquivo (RLIMIT_FSIZE) e descritores abertos (RLIMIT_NOFILE).
func applyResourceLimits(fileLimitMB, nofileLimit int) error {
	if fileLimitMB > 0 {
		limitBytes := uint64(fileLimitMB) * 1024 * 1024
		rlim := syscall.Rlimit{
			Cur: limitBytes,
			Max: limitBytes,
		}
		if err := syscall.Setrlimit(syscall.RLIMIT_FSIZE, &rlim); err != nil {
			return fmt.Errorf("failed to set RLIMIT_FSIZE (%dMB): %w", fileLimitMB, err)
		}
	}

	if nofileLimit > 0 {
		rlim := syscall.Rlimit{
			Cur: uint64(nofileLimit),
			Max: uint64(nofileLimit),
		}
		if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rlim); err != nil {
			return fmt.Errorf("failed to set RLIMIT_NOFILE (%d): %w", nofileLimit, err)
		}
	}

	return nil
}
