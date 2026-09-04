#include <stdio.h>

// Preprocessor macro exponential expansion attack
#define M1 "A"
#define M2 M1 M1 M1 M1 M1 M1 M1 M1 M1 M1
#define M3 M2 M2 M2 M2 M2 M2 M2 M2 M2 M2
#define M4 M3 M3 M3 M3 M3 M3 M3 M3 M3 M3
#define M5 M4 M4 M4 M4 M4 M4 M4 M4 M4 M4

int main() {
    printf("=== [TEST 33] Compiler Macro Expansion Bomb ===\n");
    const char *str = M5;
    printf("Result: SUCCESS - Compiled macro string length is %zu\n", sizeof(M5));
    printf("=== END TEST 33 ===\n");
    return 0;
}
