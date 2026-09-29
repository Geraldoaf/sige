// TESTE 43: Código C com erro proposital de compilação.
// Utilizado para testar se o daemon SIGE remove o workspace temporário (/workspace/exec-*)
// imediatamente após a falha do compilador gcc, impedindo o esgotamento de disco.

int main(void {  // Erro proposital de sintaxe: parêntese não fechado
    return 0;
}
