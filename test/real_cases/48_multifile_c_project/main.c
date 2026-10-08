#include <errno.h>
#include <stdio.h>
#include <string.h>
#include "include/geometry.h"
#include "include/stats.h"

int main(void) {
    /* Triangulo retangulo 3-4-5: (0,0), (3,0), (0,4) -> perimetro = 12.00 */
    Point2D triangle[3] = {
        {0.0, 0.0},
        {3.0, 0.0},
        {0.0, 4.0}
    };
    double perim = polygon_perimeter(triangle, 3);

    /* Valores: 2, 4, 4, 4, 5, 5, 7, 9 -> media = 5.00, desvio padrao = 2.00 */
    double data[8] = {2.0, 4.0, 4.0, 4.0, 5.0, 5.0, 7.0, 9.0};
    double mean = array_mean(data, 8);
    double stddev = array_stddev(data, 8);

    /* Verifica que os subdiretorios criados pelo multi-file sao somente-leitura em runtime */
    FILE *fp = fopen("/workspace/src/geometry.c", "a");
    if (fp != NULL) {
        fclose(fp);
        fprintf(stderr, "FALHA DE SEGURANCA: conseguiu abrir /workspace/src/geometry.c para escrita!\n");
        return 1;
    }
    if (errno != EROFS && errno != EACCES) {
        fprintf(stderr, "Erro inesperado ao tentar escrever em /workspace/src/geometry.c: %s\n", strerror(errno));
        return 1;
    }

    printf("PERIMETER=%.2f MEAN=%.2f STDDEV=%.2f READONLY_OK\n",
           perim, mean, stddev);
    return 0;
}
