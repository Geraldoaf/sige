package sandbox

import (
	"testing"

	seccomp "github.com/seccomp/libseccomp-golang"
)

// resolveAllowed devolve o conjunto de syscalls da allowlist que existem
// nesta arquitetura — é o conjunto que o filtro de fato libera.
func resolveAllowed(t *testing.T) map[string]bool {
	t.Helper()
	set := make(map[string]bool, len(allowedSyscalls))
	for _, name := range allowedSyscalls {
		if _, err := seccomp.GetSyscallFromName(name); err == nil {
			set[name] = true
		}
	}
	if len(set) == 0 {
		t.Fatal("allowlist resolveu para zero syscalls nesta arquitetura")
	}
	return set
}

// TestAllowlistContemEssenciais garante que a allowlist não fique estreita
// demais. Sem qualquer uma destas, nenhum programa chega a rodar — e a falha
// apareceria como "processo morto sem mensagem", que é caríssimo de
// diagnosticar em produção.
func TestAllowlistContemEssenciais(t *testing.T) {
	permitidas := resolveAllowed(t)

	essenciais := []string{
		"execve", "exit", "exit_group", "read", "write", "close", "openat",
		"mmap", "mprotect", "munmap", "brk", "rt_sigaction", "rt_sigprocmask",
		"rt_sigreturn", "futex", "set_robust_list", "set_tid_address",
		"arch_prctl", "getpid", "wait4", "newfstatat", "getrandom",
		// Servidas por vDSO no caminho normal, mas obrigatórias no fallback.
		"clock_gettime", "gettimeofday",
	}
	for _, name := range essenciais {
		if !permitidas[name] {
			t.Errorf("syscall essencial ausente da allowlist: %s", name)
		}
	}
}

// TestAllowlistNaoContemPerigosas trava as syscalls que dão acesso a
// namespaces, montagem, módulos de kernel, depuração de outros processos e
// relógio do sistema.
//
// A lista inclui deliberadamente as VARIANTES que escapavam da política
// anterior, derivada por subtração: clock_settime era negada mas
// clock_settime64 passava; quotactl era negada mas quotactl_fd passava. Com
// allowlist explícita o padrão é negar, e este teste garante que ninguém as
// reintroduza por engano.
func TestAllowlistNaoContemPerigosas(t *testing.T) {
	permitidas := resolveAllowed(t)

	proibidas := []string{
		// Namespaces e montagem
		"mount", "umount", "umount2", "pivot_root", "chroot", "unshare", "setns",
		"fsopen", "fsconfig", "fsmount", "fspick", "move_mount", "open_tree",
		"mount_setattr", "open_by_handle_at",
		// Criação de processo fora da regra condicional de clone()
		"clone3",
		// Depuração e inspeção de outros processos
		"ptrace", "process_vm_readv", "process_vm_writev", "kcmp",
		"pidfd_getfd", "process_madvise",
		// Kernel e hardware
		"init_module", "finit_module", "delete_module", "kexec_load",
		"reboot", "iopl", "ioperm", "bpf", "perf_event_open", "syslog",
		// Credenciais
		"setuid", "setgid", "setreuid", "setregid", "setresuid", "setresgid",
		"setgroups", "capset",
		// Relógio do sistema — inclui as variantes que escapavam
		"settimeofday", "clock_settime", "clock_settime64", "stime",
		"adjtimex", "clock_adjtime", "clock_adjtime64",
		// Outras que escapavam da denylist
		"quotactl", "quotactl_fd", "sethostname", "setdomainname",
		"fanotify_init", "fanotify_mark", "vhangup",
		"lsm_set_self_attr", "lsm_get_self_attr", "lsm_list_modules",
		// Keyring, io_uring e afins
		"add_key", "request_key", "keyctl",
		"io_uring_setup", "io_uring_register", "io_uring_enter",
		"userfaultfd", "uselib", "acct", "swapon", "swapoff",
	}
	for _, name := range proibidas {
		if permitidas[name] {
			t.Errorf("syscall perigosa presente na allowlist: %s", name)
		}
	}
}

// TestAllowlistSemDuplicatas evita que a lista cresça por acidente com
// entradas repetidas ao ser editada.
func TestAllowlistSemDuplicatas(t *testing.T) {
	visto := make(map[string]bool, len(allowedSyscalls))
	for _, name := range allowedSyscalls {
		if visto[name] {
			t.Errorf("entrada duplicada na allowlist: %s", name)
		}
		visto[name] = true
	}
}
