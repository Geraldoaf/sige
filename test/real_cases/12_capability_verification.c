// TESTE 12: em vez de tentar algo arriscado, só lê e imprime o próprio
// /proc/self/status — confirma diretamente que o código do usuário roda sem
// NENHUMA capability (CapEff/CapPrm/CapBnd zerados), com NoNewPrivs=1
// (setado pelo ApplySeccompFilter em internal/sandbox/seccomp.go, impede
// reganhar privilégio via exec de um binário com file capabilities — ver
// internal_launch.go) e como UID/GID 65534 (nobody).
#include <stdio.h>
#include <string.h>

int main(void) {
    printf("=== [TEST 12] Capability & NoNewPrivs Verification ===\n");
    printf("Action: Reading /proc/self/status...\n");

    FILE *f = fopen("/proc/self/status", "r");
    if (!f) {
        printf("Result: could not open /proc/self/status\n");
        return 1;
    }

    char line[256];
    while (fgets(line, sizeof(line), f)) {
        if (strncmp(line, "CapInh:", 7) == 0 ||
            strncmp(line, "CapPrm:", 7) == 0 ||
            strncmp(line, "CapEff:", 7) == 0 ||
            strncmp(line, "CapBnd:", 7) == 0 ||
            strncmp(line, "NoNewPrivs:", 11) == 0 ||
            strncmp(line, "Uid:", 4) == 0 ||
            strncmp(line, "Gid:", 4) == 0) {
            printf("%s", line);
        }
    }
    fclose(f);

    printf("Expected: CapInh/CapPrm/CapEff/CapBnd all zero, NoNewPrivs: 1, Uid/Gid all 65534.\n");
    printf("=== END TEST 12 ===\n");
    return 0;
}
