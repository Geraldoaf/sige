package cli

import (
	"fmt"
)

func Run(input string) {
	if input != "" {
		fmt.Printf("Processando entrada do terminal: %s\n", input)
	} else {
		fmt.Println("Nenhuma entrada fornecida. Use --help para ver os comandos disponíveis.")
	}
	fmt.Println("Tarefa finalizada no modo CLI.")
}
