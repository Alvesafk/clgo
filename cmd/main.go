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
	totalFilesCounted, totalLines := core.CountLinesRecursive(args[1])
	totalTime := time.Since(start).Seconds()

	fmt.Printf("Time elapsed  :: %.6f seconds.\n", totalTime)
	fmt.Printf("Files counted :: %v\nRate of Files :: %.2f/s\nRate of Lines :: %.2f/s\n",
		totalFilesCounted, float64(totalFilesCounted)/totalTime, float64(totalLines)/totalTime)
}
