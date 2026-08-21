// TESTE 20: mesma ideia dos testes 18/19, mas com /etc/shadow — diferente
// de /etc/passwd, /etc/shadow normalmente só é legível por root/grupo
// "shadow" (modo 0640). Como o compilador roda como UID 65534 (nobody)
// mesmo dentro do sandbox, o resultado esperado aqui é falha de PERMISSÃO
// ("Permission denied" no stderr do erro de compilação), não vazamento de
// conteúdo — uma segunda camada de proteção (DAC) além do isolamento de
// namespace, mesmo que o sandboxing da compilação falhasse por algum
// motivo.
#include "/etc/shadow"

int main(void) {
    return 0;
}
