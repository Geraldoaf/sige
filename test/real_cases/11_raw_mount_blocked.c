// TESTE 11: mount() está na blacklist do Stage B — sem isso, um binário
// compilado (rodando dentro do próprio pivot_root) poderia tentar remontar
// algo como read-write, ou montar um tmpfs por cima de um mount point
// existente. Mesmo com CAP_SYS_ADMIN (que o código do usuário NÃO tem, já
// que roda como UID 65534/nobody sem capabilities), o seccomp mata a
// chamada antes do kernel sequer checar a capability. Bloqueio bem-sucedido
// = processo morto na chamada, ver nota em 09_clone_namespace_flags.c.
#include <stdio.h>
#include <sys/mount.h>
#include <errno.h>
#include <string.h>

int main(void) {
    printf("=== [TEST 11] Raw mount() Syscall Block ===\n");
    printf("Action: Calling mount(\"tmpfs\", \"/tmp\", \"tmpfs\", 0, NULL)...\n");
    fflush(stdout);

    int ret = mount("tmpfs", "/tmp", "tmpfs", 0, NULL);
    printf("Result: UNEXPECTED - mount() returned %d (errno=%s), was NOT blocked!\n", ret, strerror(errno));
    printf("=== END TEST 11 ===\n");
    return 0;
}
