// TESTE 41: Disparo genuíno de SIGXFSZ pelo kernel via RLIMIT_FSIZE.
// Este programa NÃO escreve "File too large" ou "File size limit exceeded" no stderr.
// Ele apenas escreve continuamente em disco até o kernel emitir SIGXFSZ.
// O daemon deve classificar com status "file_size_exceeded" baseado unicamente no sinal do kernel.

#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <fcntl.h>

int main(void) {
    printf("=== [TEST 41] Genuine Kernel SIGXFSZ Emission ===\n");
    printf("Action: Writing to /tmp/overflow.bin beyond RLIMIT_FSIZE...\n");
    fflush(stdout);

    int fd = open("/tmp/overflow.bin", O_WRONLY | O_CREAT | O_TRUNC, 0644);
    if (fd < 0) {
        perror("open /tmp/overflow.bin");
        return 1;
    }

    char buffer[1024 * 1024]; // 1MB buffer
    for (size_t i = 0; i < sizeof(buffer); i++) {
        buffer[i] = (char)(i % 256);
    }

    // Escreve até o kernel enviar SIGXFSZ e encerrar o processo
    for (int i = 0; i < 50; i++) {
        ssize_t written = write(fd, buffer, sizeof(buffer));
        if (written < 0) {
            // Se não morreu pelo sinal e write retornou erro:
            perror("write error");
            close(fd);
            return 2;
        }
    }

    close(fd);
    printf("FAIL: Escreveu 50MB sem receber SIGXFSZ!\n");
    return 3;
}
