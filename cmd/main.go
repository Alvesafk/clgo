package main

import (
	"fmt"
	"os"
	"github.com/Alvesafk/clgo/core"
)

func main() {
	args := os.Args

	if len(args) != 2 {
		fmt.Println("WROOONG, dumbass")
		return
	}

	core.CountLinesRecursive(args[1])
}
