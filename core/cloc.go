package core

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

type FileEntry struct {
	Entry os.DirEntry
	Path  string
}

const (
	RECURSION_LIMIT = 20
)

func CountLinesRecursive(dirpath string) {
	dirs := genFileArray(getDirs(dirpath), RECURSION_LIMIT)

	jobs := make(chan FileEntry, len(dirs))
	results := make(chan int, len(dirs))

	numWorkers := runtime.NumCPU()
	var wg sync.WaitGroup

	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for v := range jobs {
				absFilename := filepath.Join(v.Path, v.Entry.Name())
				results <- countLinesOfFile(absFilename)
			}
		}()
	}

	for _, v := range dirs {
		jobs <- v
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	var totalLines int
	for r := range results {
		totalLines += r
	}

	fmt.Println(totalLines)
}
func countLinesOfFile(filename string) int {
	file, err := os.ReadFile(filename)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var counter int
	for _, c := range file {
		if c == '\n' {
			counter += 1
		}
	}

	return counter
}

func genFileArray(arr []FileEntry, recLimit int) []FileEntry {
	var dirFound int
	for i, v := range arr {
		if v.Entry.IsDir() {
			fullPath := filepath.Join(v.Path, v.Entry.Name())
			arr = append(arr, getDirs(fullPath)...)
			arr = append(arr[:i], arr[i+1:]...)
			dirFound++
		}
	}
	if dirFound > 0 && recLimit > 0 {
		arr = genFileArray(arr, recLimit-1)
	}
	return arr
}

func getDirs(dirPath string) []FileEntry {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		panic(err)
	}
	result := make([]FileEntry, 0, len(entries))
	for _, e := range entries {
		result = append(result, FileEntry{Entry: e, Path: dirPath})
	}
	return result
}
