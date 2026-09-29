// TESTE 44: Verificação de retenção de Bounding Set e Capabilities em múltiplas threads.
// Comprova que o runtime.LockOSThread() no launcher do SIGE evitou que threads do SO
// mantivessem capabilities ou bounding sets intactos antes do execve.

#include <stdio.h>
#include <pthread.h>
#include <string.h>

void *thread_check(void *arg) {
    int tid = *(int *)arg;
    FILE *f = fopen("/proc/self/status", "r");
    if (!f) {
        printf("Thread %d: could not open /proc/self/status\n", tid);
        return (void *)1;
    }

    char line[256];
    int ok = 1;
    while (fgets(line, sizeof(line), f)) {
        if (strncmp(line, "CapBnd:\t0000000000000000", 24) == 0 ||
            strncmp(line, "CapEff:\t0000000000000000", 24) == 0 ||
            strncmp(line, "CapPrm:\t0000000000000000", 24) == 0) {
            // zero como esperado
        } else if (strncmp(line, "CapBnd:", 7) == 0 ||
                   strncmp(line, "CapEff:", 7) == 0 ||
                   strncmp(line, "CapPrm:", 7) == 0) {
            printf("Thread %d VIOLATION: %s", tid, line);
            ok = 0;
        }
    }
    fclose(f);
    return (void *)(long)(ok ? 0 : 2);
}

int main(void) {
    printf("=== [TEST 44] Multithreaded Capability & Bounding Set Verification ===\n");
    pthread_t th[4];
    int ids[4] = {1, 2, 3, 4};
    int all_ok = 1;

    for (int i = 0; i < 4; i++) {
        if (pthread_create(&th[i], NULL, thread_check, &ids[i]) != 0) {
            perror("pthread_create");
            return 1;
        }
    }

    for (int i = 0; i < 4; i++) {
        void *ret;
        pthread_join(th[i], &ret);
        if ((long)ret != 0) {
            all_ok = 0;
        }
    }

    if (all_ok) {
        printf("PASS: All 4 threads have CapBnd, CapEff and CapPrm strictly zero.\n");
    } else {
        printf("FAIL: Some threads had non-zero capabilities or bounding sets!\n");
    }

    printf("=== END TEST 44 ===\n");
    return all_ok ? 0 : 1;
}
