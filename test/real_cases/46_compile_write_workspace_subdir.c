#include <stdio.h>
#include <dirent.h>
#include <errno.h>
#include <string.h>
#include <sys/stat.h>
#include <fcntl.h>
#include <unistd.h>

// TESTE 46: Verifica que o diretório /workspace após a compilação permite
// leitura/listagem (opendir) pelo usuário nobody (0755), mas permanece
// estritamente protegido contra criação de subdiretórios ou sobrescrita do
// próprio binário /workspace/solution (MS_RDONLY + 0755).

int main(void) {
    printf("=== [TEST 46] Workspace Directory Listing & Read-Only Enforcement ===\n");

    DIR *d = opendir("/workspace");
    if (!d) {
        printf("Result: UNEXPECTED - opendir(/workspace) failed: %s\n", strerror(errno));
        return 1;
    }
    int count = 0;
    struct dirent *entry;
    while ((entry = readdir(d)) != NULL) {
        count++;
    }
    closedir(d);
    printf("OK - opendir(/workspace) succeeded (%d entries visible)\n", count);

    if (mkdir("/workspace/sub_evil", 0755) == 0) {
        printf("Result: UNEXPECTED - mkdir(/workspace/sub_evil) succeeded!\n");
        return 1;
    }
    printf("OK - mkdir(/workspace/sub_evil) blocked: %s\n", strerror(errno));

    int fd = open("/workspace/solution", O_WRONLY | O_TRUNC);
    if (fd >= 0) {
        close(fd);
        printf("Result: UNEXPECTED - overwriting /workspace/solution succeeded!\n");
        return 1;
    }
    printf("OK - overwriting /workspace/solution blocked: %s\n", strerror(errno));

    printf("=== END TEST 46 ===\n");
    return 0;
}
