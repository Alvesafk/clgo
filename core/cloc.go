package core

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

type fileEntry struct {
	Entry os.DirEntry
	Path  string
}

func (f fileEntry) fullpath() string {
	return filepath.Join(f.Path, f.Entry.Name())
}

type dirResult struct {
	dirs  []fileEntry
	files []fileEntry
}

const (
	RECURSION_LIMIT = 50
)

var (
	totalFilesCounted int
)

func CountLinesRecursive(dirpath string) (int, int) {
	fileArr := make([]fileEntry, 0, 10)
	dirs := genFileArray(fileArr, getDirs(dirpath), RECURSION_LIMIT)

	jobs := make(chan fileEntry, len(dirs))
	results := make(chan int, len(dirs))

	numWorkers := runtime.NumCPU() / 2
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

	fmt.Printf("%v lines were counted on %v files.\n", totalLines, totalFilesCounted)

	return totalFilesCounted, totalLines
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

	totalFilesCounted++

	return counter
}

func genFileArray(fileArr, dirArr []fileEntry, recLimit int) []fileEntry {
	if len(dirArr) == 0 {
		return fileArr
	}

	jobs := make(chan fileEntry, len(dirArr))
	results := make(chan dirResult, len(dirArr))

	numWorkers := runtime.NumCPU() / 2

	var wg sync.WaitGroup
	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for v := range jobs {
				if v.Entry.IsDir() {
					results <- dirResult{dirs: getDirs(v.fullpath())}
				} else {
					results <- dirResult{files: []fileEntry{v}}
				}
			}
		}()
	}

	for _, v := range dirArr {
		jobs <- v
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	var nextDirArr []fileEntry
	for r := range results {
		fileArr = append(fileArr, r.files...)
		nextDirArr = append(nextDirArr, r.dirs...)
	}

	if len(nextDirArr) > 0 && recLimit > 0 {
		fileArr = genFileArray(fileArr, nextDirArr, recLimit-1)
	}

	return fileArr
}

func getDirs(dirPath string) []fileEntry {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		panic(err)
	}
	result := make([]fileEntry, 0, len(entries))
	for _, e := range entries {
		result = append(result, fileEntry{Entry: e, Path: dirPath})
	}
	return result
}
