#include <stdio.h>
#include <unistd.h>

int main(void) {
    pid_t pid = getpid();
    printf("=== [TEST 45] PID Namespace Verification ===\n");
    printf("PID: %d\n", pid);

    if (pid == 1) {
        printf("FAIL: Running as unmanaged PID 1 in PID namespace\n");
        return 1;
    }

    printf("PASS: Running as PID %d (> 1) under mini-init supervisor\n", pid);
    return 0;
}
