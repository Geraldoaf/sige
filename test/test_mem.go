package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("Iniciando alocação de memória...")
	var data [][]byte
	for {

		chunk := make([]byte, 10*1024*1024)
		for i := range chunk {
			chunk[i] = 1
		}
		data = append(data, chunk)
		fmt.Printf("Alocados: %d MB\n", len(data)*10)
		time.Sleep(500 * time.Millisecond)
	}
}
