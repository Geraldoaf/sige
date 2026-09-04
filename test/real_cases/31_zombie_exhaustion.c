#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/types.h>

int main() {
    printf("=== [TEST 31] Zombie Process Exhaustion ===\n");
    printf("Action: Rapidly creating child processes without reaping to test PID limits...\n");

    int created = 0;
    for (int i = 0; i < 200; i++) {
        pid_t pid = fork();
        if (pid < 0) {
            printf("Result: BLOCKED - fork failed at child #%d (pids limit reached)\n", i);
            break;
        }
        if (pid == 0) {
            // Child exits immediately becoming a zombie until parent reaps
            exit(0);
        }
        created++;
    }

    printf("Action: Created %d zombie children. Exiting parent to test PID namespace auto-reap...\n", created);
    printf("=== END TEST 31 ===\n");
    return 0;
}
