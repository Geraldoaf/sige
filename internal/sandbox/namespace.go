package sandbox

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"sige/internal/constants"
)

func ConfigureSandboxNamespace(rootfs, workspace string, tmpLimitMB, fileLimitMB, nofileLimit int) error {

	if err := syscall.Mount("", "/", "", syscall.MS_REC|syscall.MS_PRIVATE, ""); err != nil {
		return fmt.Errorf("error configuring mount propagation as private: %w", err)
	}

	if err := syscall.Mount(rootfs, rootfs, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("error bind-mounting temporary rootfs: %w", err)
	}

	dirsToMount := []string{"/bin", "/lib", "/lib64", "/usr", "/etc"}
	for _, dir := range dirsToMount {
		if _, err := os.Stat(dir); err != nil {
			if os.IsNotExist(err) {
				continue
			}
		}

		dest := filepath.Join(rootfs, dir)
		if err := os.MkdirAll(dest, 0755); err != nil {
			return fmt.Errorf("error creating mount point %s: %w", dest, err)
		}

		if err := syscall.Mount(dir, dest, "", syscall.MS_BIND|syscall.MS_RDONLY, ""); err != nil {
			return fmt.Errorf("error mounting %s to %s: %w", dir, dest, err)
		}

		_ = syscall.Mount("", dest, "", syscall.MS_BIND|syscall.MS_RDONLY|syscall.MS_REMOUNT, "")
	}

	procDest := filepath.Join(rootfs, "proc")
	if err := os.MkdirAll(procDest, 0755); err != nil {
		return fmt.Errorf("error creating proc directory in rootfs: %w", err)
	}
	if err := syscall.Mount("proc", procDest, "proc", 0, "hidepid=2"); err != nil {
		return fmt.Errorf("error mounting proc: %w", err)
	}

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

	workspaceDest := filepath.Join(rootfs, "workspace")
	if err := os.MkdirAll(workspaceDest, 0755); err != nil {
		return fmt.Errorf("error creating workspace folder in rootfs: %w", err)
	}
	if err := syscall.Mount(workspace, workspaceDest, "", syscall.MS_BIND|syscall.MS_RDONLY, ""); err != nil {
		return fmt.Errorf("error mounting workspace to %s: %w", workspaceDest, err)
	}

	_ = syscall.Mount("", workspaceDest, "", syscall.MS_BIND|syscall.MS_RDONLY|syscall.MS_REMOUNT, "")

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

	if err := syscall.Chdir("/workspace"); err != nil {
		return fmt.Errorf("error changing directory to /workspace: %w", err)
	}

	if err := applyResourceLimits(fileLimitMB, nofileLimit); err != nil {
		return fmt.Errorf("error applying resource limits (rlimits): %w", err)
	}

	if err := syscall.Setgid(constants.DefaultGIDNobody); err != nil {
		return fmt.Errorf("error dropping GID privileges to nobody: %w", err)
	}
	if err := syscall.Setuid(constants.DefaultUIDNobody); err != nil {
		return fmt.Errorf("error dropping UID privileges to nobody: %w", err)
	}

	return nil
}

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
