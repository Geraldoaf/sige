package sandbox

import (
	"fmt"

	seccomp "github.com/seccomp/libseccomp-golang"
)

func ApplySeccompFilter() error {

	filter, err := seccomp.NewFilter(seccomp.ActAllow)
	if err != nil {
		return fmt.Errorf("error creating seccomp filter: %w", err)
	}
	defer filter.Release()

	blockedSyscalls := []string{

		"unshare", "setns", "pivot_root", "mount", "umount", "umount2", "chroot", "open_by_handle_at",

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

	for _, name := range blockedSyscalls {
		syscallID, err := seccomp.GetSyscallFromName(name)
		if err != nil {

			continue
		}
		if err := filter.AddRule(syscallID, seccomp.ActKillProcess); err != nil {
			return fmt.Errorf("error blocking syscall %s: %w", name, err)
		}
	}

	return filter.Load()
}
