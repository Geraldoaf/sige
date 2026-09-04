#include <stdio.h>
#include <stdlib.h>
#include <sys/ipc.h>
#include <sys/shm.h>
#include <sys/msg.h>
#include <sys/sem.h>
#include <sys/syscall.h>
#include <unistd.h>
#include <errno.h>

int main() {
    printf("=== [TEST 30] IPC & Kernel Keyring Isolation ===\n");
    printf("Action: Probing SysV IPC shared memory, message queues and keyctl...\n");

    // Try creating/accessing shared memory
    int shmid = shmget(IPC_PRIVATE, 1024, IPC_CREAT | 0666);
    if (shmid < 0) {
        printf("Result: shmget failed (%d)\n", errno);
    } else {
        printf("OK - shmget isolated within local namespace (id=%d)\n", shmid);
        shmctl(shmid, IPC_RMID, NULL);
    }

    // Try keyctl syscall (should be blocked by seccomp)
    #ifdef __NR_keyctl
    long res = syscall(__NR_keyctl, 0 /* KEYCTL_GET_KEYRING_ID */, -1, 0, 0, 0);
    printf("Result: UNEXPECTED - keyctl syscall succeeded or returned %ld\n", res);
    #else
    printf("OK - keyctl syscall not defined in architecture\n");
    #endif

    printf("=== END TEST 30 ===\n");
    return 0;
}
