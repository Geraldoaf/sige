// TESTE 09: chama clone() diretamente com CLONE_NEWUSER — ao contrário dos
// outros namespaces, criar um user namespace não exige nenhuma capability,
// então é a forma mais simples de um processo comum tentar "virar root"
// (dentro do namespace novo). unshare()/setns() já eram bloqueados; clone3
// também. Este teste verifica a regra condicional adicionada em
// internal/sandbox/seccomp.go (blockCloneWithNamespaceFlags) que mata o
// processo se QUALQUER flag CLONE_NEW* estiver no primeiro argumento do
// clone() "clássico" — sem bloquear clone() em geral, que fork()/threads de
// qualquer programa legítimo precisam.
//
// IMPORTANTE: se a regra funcionar, o processo é morto pelo kernel
// (SECCOMP_RET_KILL_PROCESS) na hora da chamada — não é um erro que dá pra
// capturar. Ou seja, um bloqueio bem-sucedido aparece como: stdout para
// exatamente depois da linha "Action: ...", e o exit_code/status da
// execução (na resposta da API) indica término por sinal (não um retorno
// normal). As linhas de "Result:" abaixo só aparecem se o bloqueio FALHOU.
#define _GNU_SOURCE
#include <stdio.h>
#include <sched.h>
#include <signal.h>
#include <errno.h>
#include <string.h>
#include <sys/wait.h>

static char child_stack[1024 * 1024];

static int child_fn(void *arg) {
    printf("Result: UNEXPECTED - child running inside a freshly created user namespace!\n");
    fflush(stdout);
    return 0;
}

int main(void) {
    printf("=== [TEST 09] clone() with CLONE_NEWUSER (namespace escape attempt) ===\n");
    printf("Action: Calling clone(CLONE_NEWUSER | SIGCHLD, ...)...\n");
    fflush(stdout);

    pid_t pid = clone(child_fn, child_stack + sizeof(child_stack), CLONE_NEWUSER | SIGCHLD, NULL);
    if (pid == -1) {
        printf("Result: BLOCKED (denied by kernel/capability check, not seccomp) - clone() returned -1: %s\n", strerror(errno));
        printf("=== END TEST 09 ===\n");
        return 0;
    }

    int status;
    waitpid(pid, &status, 0);
    printf("Result: clone() succeeded, child pid=%d\n", pid);
    printf("=== END TEST 09 ===\n");
    return 0;
}
