package sandbox

import (
	"fmt"
	"syscall"

	seccomp "github.com/seccomp/libseccomp-golang"
)

// deniedSyscalls são as syscalls explicitamente negadas para os comandos executados no sandbox.
var deniedSyscalls = []string{
	"unshare", "setns", "pivot_root", "mount", "umount", "umount2", "chroot", "open_by_handle_at",
	"clone3", "socketcall",
	"fsopen", "fsconfig", "fsmount", "fspick", "move_mount", "open_tree", "mount_setattr",
	"ptrace", "process_vm_readv", "process_vm_writev", "kcmp",
	"kexec_load", "kexec_file_load", "reboot", "syslog",
	"init_module", "finit_module", "delete_module",
	"acct", "swapon", "swapoff",
	"setuid", "setgid", "setreuid", "setregid",
	"setresuid", "setresgid", "setfsuid", "setfsgid",
	"setgroups", "capset",
	"bpf", "perf_event_open", "quotactl",
	"settimeofday", "clock_settime", "adjtimex", "clock_adjtime",
	"nfsservctl", "personality", "lookup_dcookie",
	"iopl", "ioperm", "create_module", "get_kernel_syms", "query_module",
	"add_key", "request_key", "keyctl",
	"io_uring_setup", "io_uring_register", "io_uring_enter",
	"userfaultfd", "uselib", "vm86", "vm86old",
}

// namespaceCloneFlags agrupa flags de isolamento de namespace para checagem condicional em clone(2).
var namespaceCloneFlags = []uintptr{
	syscall.CLONE_NEWNS,
	syscall.CLONE_NEWUTS,
	syscall.CLONE_NEWIPC,
	syscall.CLONE_NEWUSER,
	syscall.CLONE_NEWPID,
	syscall.CLONE_NEWNET,
	syscall.CLONE_NEWCGROUP,
}

// ApplySeccompFilter instala o filtro Seccomp rigoroso antes da execução do código do usuário.
func ApplySeccompFilter() error {
	filter, err := seccomp.NewFilter(seccomp.ActKillProcess)
	if err != nil {
		return fmt.Errorf("error creating seccomp filter: %w", err)
	}
	defer filter.Release()

	baseNames, err := baselineSyscallNames()
	if err != nil {
		return err
	}

	denied := make(map[string]bool, len(deniedSyscalls))
	for _, name := range deniedSyscalls {
		denied[name] = true
	}

	allowed := 0
	for _, name := range baseNames {
		if denied[name] {
			continue
		}
		if name == "clone" {
			continue
		}

		syscallID, err := seccomp.GetSyscallFromName(name)
		if err != nil {
			continue
		}
		if err := filter.AddRule(syscallID, seccomp.ActAllow); err != nil {
			return fmt.Errorf("error allowing syscall %s: %w", name, err)
		}
		allowed++
	}

	if allowed == 0 {
		return fmt.Errorf("seccomp allowlist resolved to zero allowed syscalls")
	}

	if err := allowCloneWithoutNamespaceFlags(filter); err != nil {
		return err
	}

	if err := denyClone3WithENOSYS(filter); err != nil {
		return err
	}

	return filter.Load()
}

// denyClone3WithENOSYS faz clone3 retornar ENOSYS para permitir fallback gracioso na glibc.
func denyClone3WithENOSYS(filter *seccomp.ScmpFilter) error {
	clone3ID, err := seccomp.GetSyscallFromName("clone3")
	if err != nil {
		return nil
	}

	const enosys = 38
	if err := filter.AddRule(clone3ID, seccomp.ActErrno.SetReturnCode(enosys)); err != nil {
		return fmt.Errorf("error setting clone3 to return ENOSYS: %w", err)
	}
	return nil
}

// allowCloneWithoutNamespaceFlags permite chamadas clone(2) que não solicitem criação de novos namespaces.
func allowCloneWithoutNamespaceFlags(filter *seccomp.ScmpFilter) error {
	cloneID, err := seccomp.GetSyscallFromName("clone")
	if err != nil {
		return nil
	}

	var mask uint64
	for _, flag := range namespaceCloneFlags {
		mask |= uint64(flag)
	}

	cond, err := seccomp.MakeCondition(0, seccomp.CompareMaskedEqual, mask, 0)
	if err != nil {
		return fmt.Errorf("error building clone() namespace-flag condition: %w", err)
	}

	if err := filter.AddRuleConditional(cloneID, seccomp.ActAllow, []seccomp.ScmpCondition{cond}); err != nil {
		return fmt.Errorf("error allowing clone() without namespace flags: %w", err)
	}

	return nil
}
