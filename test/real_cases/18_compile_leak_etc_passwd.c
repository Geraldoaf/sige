// TESTE 18 — o achado mais crítico da auditoria de segurança (SIGE-001):
// antes, a compilação C/C++ rodava DIRETO no processo da API (fora de
// qualquer sandbox). #include de um arquivo arbitrário do host fazia o GCC
// ler esse arquivo, e o erro de compilação resultante (que a API devolve
// pro cliente) vazava o conteúdo pelo stderr — um primitivo de leitura de
// arquivo do servidor real, não do sandbox.
//
// Depois do fix (internal/api/workspace.go: compileSource agora chama
// sandbox.Execute(), não exec.CommandContext direto), o compilador roda
// dentro do MESMO isolamento (namespaces + pivot_root) de qualquer
// execução — ou seja, só pode "vazar" o que Python/Bash já enxergariam de
// qualquer forma dentro do sandbox, nunca mais o ambiente real da API.
//
// /etc/passwd é usado aqui (em vez de /etc/shadow) porque costuma ser
// legível por qualquer UID — então este teste É esperado mostrar conteúdo
// no stderr do erro de compilação. O que importa verificar é QUAL conteúdo:
// deve ser o /etc/passwd do CONTAINER (as mesmas poucas linhas que
// 05_write_rootfs.py e qualquer outra execução já veriam), nunca nada do
// processo da API em si.
#include "/etc/passwd"

int main(void) {
    return 0;
}
