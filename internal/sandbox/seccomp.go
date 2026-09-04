package sandbox

import (
	"fmt"
	"syscall"

	seccomp "github.com/seccomp/libseccomp-golang"
)

// allowedSyscalls é a lista EXPLÍCITA de syscalls disponíveis ao código do
// usuário. Tudo que não está aqui é morto pela ação padrão do filtro.
//
// A lista foi levantada empiricamente: cada workload suportado (python3,
// bash, gcc, g++ e binários C/C++ compilados) foi tracejado com strace
// dentro da própria imagem de produção, e a união do que usam forma a base.
// A ela somam-se três grupos que o tracing não consegue revelar:
//
//   - chamadas servidas por vDSO (clock_gettime, gettimeofday, time...), que
//     não entram no kernel e portanto não aparecem no strace, mas precisam
//     ser permitidas para o caso de fallback para a syscall real;
//   - exit/exit_group, que o strace -c não contabiliza;
//   - variantes por arquitetura e alternativas que a libc pode escolher
//     conforme a versão (open/openat, stat/newfstatat/statx, ...).
//
// Antes esta política era derivada por SUBTRAÇÃO do perfil base (todas as
// syscalls do perfil menos uma denylist), o que tinha dois defeitos: syscalls
// novas do kernel entravam permitidas sem revisão, e variantes escapavam da
// denylist — clock_settime era negada mas clock_settime64 passava, quotactl
// era negada mas quotactl_fd passava. Com allowlist explícita, o padrão para
// qualquer coisa não enumerada é morrer.
var allowedSyscalls = []string{
	// Processo e execução
	"arch_prctl", "execve", "execveat", "exit", "exit_group", "fork",
	"get_robust_list", "getpgrp", "getpid", "getppid", "getsid",
	"gettid", "prctl", "rseq", "set_robust_list", "set_tid_address",
	"setsid", "vfork", "wait4", "waitid",

	// Sinais
	"kill", "pause", "restart_syscall", "rt_sigaction", "rt_sigpending",
	"rt_sigprocmask", "rt_sigqueueinfo", "rt_sigreturn",
	"rt_sigsuspend", "rt_sigtimedwait", "sigaltstack", "tgkill",
	"tkill",

	// Memória
	"brk", "madvise", "memfd_create", "mmap", "mmap2", "mprotect",
	"mremap", "msync", "munmap",

	// Arquivos — abrir/fechar/ler/escrever
	"_llseek", "close", "close_range", "dup", "dup2", "dup3", "lseek",
	"open", "openat", "openat2", "pipe", "pipe2", "pread64", "preadv",
	"pwrite64", "pwritev", "read", "readv", "write", "writev",

	// Arquivos — metadados e diretórios
	"access", "chdir", "chmod", "copy_file_range", "faccessat",
	"faccessat2", "fadvise64", "fadvise64_64", "fallocate", "fchdir",
	"fchmod", "fchmodat", "fcntl", "fcntl64", "fdatasync", "flock",
	"fstat", "fstat64", "fstatat64", "fstatfs", "fsync", "ftruncate",
	"ftruncate64", "futimesat", "getcwd", "getdents", "getdents64",
	"link", "linkat", "lstat", "lstat64", "mkdir", "mkdirat",
	"newfstatat", "readlink", "readlinkat", "rename", "renameat",
	"renameat2", "rmdir", "sendfile", "sendfile64", "stat", "stat64",
	"statfs", "statx", "symlink", "symlinkat", "truncate", "umask",
	"unlink", "unlinkat", "utimensat",

	// Multiplexação de E/S
	"_newselect", "epoll_create", "epoll_create1", "epoll_ctl",
	"epoll_pwait", "epoll_pwait2", "epoll_wait", "eventfd", "eventfd2",
	"poll", "ppoll", "pselect6", "select",

	// Threads e sincronização
	"futex", "futex_time64", "futex_waitv", "get_thread_area",
	"membarrier", "sched_get_priority_max", "sched_get_priority_min",
	"sched_getaffinity", "sched_getparam", "sched_getscheduler",
	"sched_yield", "set_thread_area",

	// Sockets (namespace de rede vazio — sem alcance externo)
	"accept", "accept4", "bind", "connect", "getpeername",
	"getsockname", "getsockopt", "listen", "recvfrom", "recvmsg",
	"sendmsg", "sendto", "setsockopt", "shutdown", "socket",
	"socketpair",

	// Tempo (inclui os que normalmente vêm por vDSO)
	"clock_getres", "clock_gettime", "clock_gettime64",
	"clock_nanosleep", "getcpu", "gettimeofday", "nanosleep", "time",
	"times",

	// Identidade e limites (somente leitura)
	"getegid", "geteuid", "getgid", "getgroups", "getrandom",
	"getresgid", "getresuid", "getrlimit", "getrusage", "getuid",
	"prlimit64", "setrlimit", "sysinfo", "ugetrlimit", "uname",

	// Diversos necessários a runtimes
	"ioctl",
}

// namespaceCloneFlags são as flags CLONE_NEW* que, no primeiro argumento de
// clone(2), criariam um namespace novo.
var namespaceCloneFlags = []uintptr{
	syscall.CLONE_NEWNS,
	syscall.CLONE_NEWUTS,
	syscall.CLONE_NEWIPC,
	syscall.CLONE_NEWUSER,
	syscall.CLONE_NEWPID,
	syscall.CLONE_NEWNET,
	syscall.CLONE_NEWCGROUP,
}

// ApplySeccompFilter instala o filtro aplicado ao código do usuário,
// imediatamente antes do exec (ver cmd/internal_launch.go). A ação padrão é
// matar o processo; só o que está em allowedSyscalls passa.
func ApplySeccompFilter() error {
	filter, err := seccomp.NewFilter(seccomp.ActKillProcess)
	if err != nil {
		return fmt.Errorf("error creating seccomp filter: %w", err)
	}
	defer filter.Release()

	permitidas := 0
	for _, name := range allowedSyscalls {
		id, err := seccomp.GetSyscallFromName(name)
		if err != nil {
			// Syscall inexistente nesta arquitetura/kernel: as variantes de
			// 32 bits, por exemplo, não resolvem em x86-64.
			continue
		}
		if err := filter.AddRule(id, seccomp.ActAllow); err != nil {
			return fmt.Errorf("error allowing syscall %s: %w", name, err)
		}
		permitidas++
	}
	if permitidas == 0 {
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

// allowCloneWithoutNamespaceFlags permite clone(2) apenas quando NENHUMA flag
// CLONE_NEW* está presente: fork() e criação de threads passam, criar
// namespace novo cai na ação padrão e mata o processo.
//
// CLONE_NEWUSER é o caso mais relevante — criar user namespace não exige
// capability alguma, sendo a única via pela qual um processo sem privilégio
// alcançaria superfície de kernel que de outra forma não alcança.
func allowCloneWithoutNamespaceFlags(filter *seccomp.ScmpFilter) error {
	id, err := seccomp.GetSyscallFromName("clone")
	if err != nil {
		return nil
	}

	var mask uint64
	for _, flag := range namespaceCloneFlags {
		mask |= uint64(flag)
	}

	// (arg0 & mask) == 0 -> nenhuma flag de namespace pedida -> permite.
	cond, err := seccomp.MakeCondition(0, seccomp.CompareMaskedEqual, mask, 0)
	if err != nil {
		return fmt.Errorf("error building clone() namespace-flag condition: %w", err)
	}
	if err := filter.AddRuleConditional(id, seccomp.ActAllow, []seccomp.ScmpCondition{cond}); err != nil {
		return fmt.Errorf("error allowing clone() without namespace flags: %w", err)
	}
	return nil
}

// denyClone3WithENOSYS faz clone3(2) retornar ENOSYS em vez de matar o
// processo. Continua indisponível — muda apenas COMO é negada, e isso é
// obrigatório: a glibc 2.34+ usa clone3 no pthread_create e só cai para
// clone() ao receber ENOSYS. Matando, o fallback nunca ocorre e qualquer
// programa que criasse uma thread morria (ver test/real_cases/26).
func denyClone3WithENOSYS(filter *seccomp.ScmpFilter) error {
	id, err := seccomp.GetSyscallFromName("clone3")
	if err != nil {
		return nil
	}
	const enosys = 38
	if err := filter.AddRule(id, seccomp.ActErrno.SetReturnCode(enosys)); err != nil {
		return fmt.Errorf("error setting clone3 to return ENOSYS: %w", err)
	}
	return nil
}
