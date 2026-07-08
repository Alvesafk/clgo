package core

import (
	"fmt"
	"os"
)

func CountLinesOfFile(filename string) {
	file, err := os.ReadFile(filename)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var counter int
	for _, c := range file {
		if c == '\n' {
			counter+=1
		}
	}

	fmt.Println(counter)
}
