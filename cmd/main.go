package main

import (
	"fmt"
	"os"
	"time"

	"github.com/Alvesafk/clgo/core"
)

func main() {
	args := os.Args

	if len(args) != 2 {
		fmt.Println("WROOONG, dumbass")
		return
	}

	start := time.Now()
	core.CountLinesRecursive(args[1])
	fmt.Printf("Time elapsed: %.6f\n", time.Since(start).Seconds())
}
