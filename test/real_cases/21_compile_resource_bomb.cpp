// TESTE 21: "bomba de compilação" — código-fonte pequeno que expande
// exponencialmente via macro do pré-processador (padrão clássico), gerando
// uma expressão aritmética válida com ~2^N termos que o G++ precisa
// tokenizar, expandir e fazer constant-folding. Antes do fix, a compilação
// rodava sem NENHUM limite de memória/CPU além de um timeout fixo de 10s no
// processo da API — um código assim podia consumir memória livremente no
// host. Agora a compilação roda dentro do sandbox com limites fixos e
// não-configuráveis pelo cliente (ver constants.DefaultCompile* em
// internal/constants/constants.go: 256MB, 100% CPU, 10s, 64MB de /tmp).
//
// N=22 (2^22, ~4 milhões de termos "1+1+...") já é sensivelmente mais lento
// que compilar um programa normal aqui em máquina de desenvolvimento — em
// hardware mais fraco ou com N maior (A24, A26...) espera-se estourar o
// timeout de 10s e/ou o limite de 256MB antes de terminar. Resultado
// esperado: "compilation_error" com mensagem de timeout/memória — nunca o
// processo da API travando ou sendo derrubado.
#define A0 1
#define A1 A0+A0
#define A2 A1+A1
#define A3 A2+A2
#define A4 A3+A3
#define A5 A4+A4
#define A6 A5+A5
#define A7 A6+A6
#define A8 A7+A7
#define A9 A8+A8
#define A10 A9+A9
#define A11 A10+A10
#define A12 A11+A11
#define A13 A12+A12
#define A14 A13+A13
#define A15 A14+A14
#define A16 A15+A15
#define A17 A16+A16
#define A18 A17+A17
#define A19 A18+A18
#define A20 A19+A19
#define A21 A20+A20
#define A22 A21+A21

constexpr long long bomb_value = A22;

int main() {
    return static_cast<int>(bomb_value % 2);
}
