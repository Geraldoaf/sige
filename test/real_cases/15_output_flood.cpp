// TESTE 15: mesmo estouro de output do 14_output_flood.py, mas num binário
// compilado — confirma que o limite de stdout/stderr (internal/sandbox/executor.go)
// funciona igual independente da linguagem, já que ele intercepta o pipe do
// processo, não algo específico do interpretador.
#include <iostream>
#include <string>

int main() {
    std::cout << "=== [TEST 15] Stdout Flood in Compiled C++ (1MB output cap) ===\n";
    std::cout << "Action: Printing indefinitely to exceed the output limit...\n";
    std::cout.flush();

    std::string chunk(65536, 'B');
    while (true) {
        std::cout << chunk << std::flush;
    }
    return 0;
}
