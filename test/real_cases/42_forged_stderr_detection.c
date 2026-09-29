// TESTE 42: Tentativa adversária de forjar o veredito "file_size_exceeded".
// O programa imprime deliberadamente no stderr as strings de erro de estouro
// de arquivo e sai com exit code 1.
// O daemon corrigido NÃO deve cair nessa armadilha — o status deve ser "failed"
// com exit_code 1, e NUNCA "file_size_exceeded".

#include <stdio.h>
#include <stdlib.h>

int main(void) {
    printf("=== [TEST 42] Forged Stderr Attack Verification ===\n");
    printf("Action: Emitting forged file size error strings to stderr and exiting with 1...\n");
    fflush(stdout);

    // Texto que enganava a heurística antiga no daemon:
    fprintf(stderr, "File too large\n");
    fprintf(stderr, "File size limit exceeded\n");
    fflush(stderr);

    // Termina normalmente com exit code 1 (sem sinal do kernel)
    return 1;
}
