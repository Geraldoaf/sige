// TESTE 40: Tenta instanciar soquetes com famílias proibidas pelo seccomp
// (AF_VSOCK = 40 e AF_ALG = 38).
// No modelo seguro, o seccomp bloqueia essas chamadas impedindo que o código
// alcance a API de criptografia do kernel ou serviços do hipervisor via vsock.

#include <stdio.h>
#include <sys/socket.h>
#include <errno.h>
#include <string.h>

#ifndef AF_VSOCK
#define AF_VSOCK 40
#endif

#ifndef AF_ALG
#define AF_ALG 38
#endif

#ifndef AF_NETLINK
#define AF_NETLINK 16
#endif

int main(void) {
    printf("=== [TEST 40] Blocked Socket Families (AF_VSOCK, AF_ALG, AF_NETLINK) ===\n");
    int failed = 0;

    // 1. Tenta AF_VSOCK
    errno = 0;
    int s_vsock = socket(AF_VSOCK, SOCK_STREAM, 0);
    if (s_vsock >= 0) {
        printf("FAIL: socket(AF_VSOCK) foi permitido inesperadamente!\n");
        failed = 1;
    } else {
        printf("PASS: socket(AF_VSOCK) bloqueado com errno=%d (%s)\n", errno, strerror(errno));
    }

    // 2. Tenta AF_ALG
    errno = 0;
    int s_alg = socket(AF_ALG, SOCK_SEQPACKET, 0);
    if (s_alg >= 0) {
        printf("FAIL: socket(AF_ALG) foi permitido inesperadamente!\n");
        failed = 1;
    } else {
        printf("PASS: socket(AF_ALG) bloqueado com errno=%d (%s)\n", errno, strerror(errno));
    }

    // 3. Tenta AF_NETLINK
    errno = 0;
    int s_nl = socket(AF_NETLINK, SOCK_RAW, 0);
    if (s_nl >= 0) {
        printf("FAIL: socket(AF_NETLINK) foi permitido inesperadamente!\n");
        failed = 1;
    } else {
        printf("PASS: socket(AF_NETLINK) bloqueado com errno=%d (%s)\n", errno, strerror(errno));
    }

    // 4. AF_UNIX deve funcionar normalmente para IPC legítimo
    errno = 0;
    int s_unix = socket(AF_UNIX, SOCK_STREAM, 0);
    if (s_unix >= 0) {
        printf("PASS: socket(AF_UNIX) permitido para IPC legitimo\n");
    } else {
        printf("INFO: socket(AF_UNIX) retornou errno=%d (%s)\n", errno, strerror(errno));
    }

    printf("=== END TEST 40 ===\n");
    return failed ? 1 : 0;
}
