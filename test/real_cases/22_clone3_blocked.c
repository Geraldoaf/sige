// TESTE 22: clone3 (Linux 5.3+) é negada porque alcança a mesma criação de
// namespace de clone()/unshare() por uma syscall diferente.
//
// Diferente das outras syscalls negadas, esta NÃO mata o processo: retorna
// ENOSYS. A razão é funcional, não de segurança — a glibc 2.34+ usa clone3
// como caminho preferido do pthread_create e só cai para clone() ao receber
// ENOSYS. Matando o processo, o fallback nunca acontecia e QUALQUER programa
// que criasse uma thread morria (ver caso 26). A syscall continua
// igualmente inacessível; ver denyClone3WithENOSYS em
// internal/sandbox/seccomp.go.
//
// Portanto o resultado esperado aqui é o programa rodar até o fim e
// imprimir ENOSYS — não ser morto.
#define _GNU_SOURCE
#include <stdio.h>
#include <unistd.h>
#include <sys/syscall.h>
#include <errno.h>
#include <string.h>

int main(void) {
    printf("=== [TEST 22] clone3() Syscall Block ===\n");
    printf("Action: Calling raw syscall(SYS_clone3, ...)...\n");
    fflush(stdout);

    // struct clone_args zerada (88 bytes cobre as versões de kernel atuais;
    // não incluímos <linux/sched.h> pra evitar conflito com <sched.h>).
    unsigned char cl_args[88] = {0};
    errno = 0;
    long ret = syscall(SYS_clone3, cl_args, sizeof(cl_args));

    if (ret == -1 && errno == ENOSYS) {
        printf("Result: BLOCKED - clone3() returned ENOSYS (permite o fallback da glibc)\n");
    } else if (ret == -1) {
        printf("Result: BLOCKED - clone3() falhou com %s\n", strerror(errno));
    } else {
        printf("Result: UNEXPECTED - clone3() returned %ld, was NOT blocked!\n", ret);
    }
    printf("=== END TEST 22 ===\n");
    return 0;
}
