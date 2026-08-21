// TESTE 19: mesmo ataque do 18_compile_leak_etc_passwd.c, mas com g++, pra
// confirmar que o fix de compilação sandboxada vale pros dois compiladores
// (compileSource trata "c" e "cpp"/"c++" pelo mesmo caminho).
#include "/etc/passwd"

int main() {
    return 0;
}
