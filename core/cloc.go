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

func (f FileEntry) fullpath() string {
	return filepath.Join(f.Path, f.Entry.Name())
}

const (
	RECURSION_LIMIT = 20
)

var (
	totalFiles int
)

func CountLinesRecursive(dirpath string) {
	fileArr := make([]FileEntry, 0, 10)
	dirs := genFileArray(fileArr, getDirs(dirpath), RECURSION_LIMIT)

	jobs := make(chan FileEntry, len(dirs))
	results := make(chan int, len(dirs))

	numWorkers := runtime.NumCPU()
	var wg sync.WaitGroup

	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for v := range jobs {
				results <- countLinesOfFile(v.fullpath())
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

	fmt.Printf("%v lines were counted on %v files.\n", totalLines, totalFiles)
}

func countLinesOfFile(filename string) int {
	fi, err := os.Stat(filename)
	if err != nil {
		fmt.Println(err)
		return 0
	}

	if fi.Mode().IsDir() {
		return 0
	}

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

	totalFiles++

	return counter
}

func genFileArray(fileArr, dirArr []FileEntry, recLimit int) []FileEntry {
	var nextDirArr []FileEntry

	for _, v := range dirArr {
		if v.Entry.IsDir() {
			nextDirArr = append(nextDirArr, getDirs(v.fullpath())...)
		} else {
			fileArr = append(fileArr, v)
		}
	}

	if len(nextDirArr) > 0 && recLimit > 0 {
		fileArr = genFileArray(fileArr, nextDirArr, recLimit-1)
	}

	return fileArr
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
