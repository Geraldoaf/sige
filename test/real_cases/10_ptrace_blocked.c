// TESTE 10: ptrace() está na blacklist do Stage B (internal/sandbox/seccomp.go)
// — bloqueia inspeção/injeção em outros processos (inclusive PTRACE_TRACEME,
// usado aqui, que é o primeiro passo clássico de vários exploits de debug/
// injeção). Um bloqueio bem-sucedido mata o processo na hora da chamada —
// ver nota em 09_clone_namespace_flags.c sobre como interpretar isso.
#include <stdio.h>
#include <sys/ptrace.h>
#include <errno.h>
#include <string.h>

int main(void) {
    printf("=== [TEST 10] ptrace() Syscall Block ===\n");
    printf("Action: Calling ptrace(PTRACE_TRACEME, 0, NULL, NULL)...\n");
    fflush(stdout);

    long ret = ptrace(PTRACE_TRACEME, 0, NULL, NULL);
    printf("Result: UNEXPECTED - ptrace() returned %ld (errno=%s), was NOT blocked!\n", ret, strerror(errno));
    printf("=== END TEST 10 ===\n");
    return 0;
}
