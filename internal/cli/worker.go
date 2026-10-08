package cli

import (
	"fmt"
)

// Run processa a execução de tarefas diretamente pelo terminal.
func Run(input string) {
	if input != "" {
		fmt.Printf("Processing terminal input: %s\n", input)
	} else {
		fmt.Println("No input provided. Use --help to view available commands.")
	}
	fmt.Println("Task finished in CLI mode.")
}
