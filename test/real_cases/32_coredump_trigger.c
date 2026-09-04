#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <signal.h>
#include <sys/resource.h>

int main() {
    printf("=== [TEST 32] Core Dump & Crash Handling ===\n");
    printf("Action: Checking core dump limit and triggering SIGSEGV...\n");

    struct rlimit rl;
    if (getrlimit(RLIMIT_CORE, &rl) == 0) {
        printf("RLIMIT_CORE cur=%lu max=%lu\n", rl.rlim_cur, rl.rlim_max);
    }

    // Allocate dirty memory
    char *buf = malloc(10 * 1024 * 1024);
    if (buf) {
        memset(buf, 0x41, 10 * 1024 * 1024);
    }

    // Trigger intentional SIGSEGV
    volatile int *null_ptr = NULL;
    *null_ptr = 42;

    printf("Result: UNEXPECTED - Reached code after null pointer dereference!\n");
    return 0;
}
