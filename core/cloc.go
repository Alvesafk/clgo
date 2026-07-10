/*
Copyright (c) 2026 Alvesafk. All Rights Reserved.

Core package has the business logic of clgo.
*/
package core

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// Config struct for optional flags defined in main.go.
type Config struct {
	NoStats          bool
	NoIgnoreDotFiles bool
	Recursion        int
}

// fileEntry struct has the actual os.DirEntry and the path of the file.
type fileEntry struct {
	Entry os.DirEntry
	Path  string
}

// fullpath method retuns a string with the full path of f file.
func (f fileEntry) fullpath() string {
	return filepath.Join(f.Path, f.Entry.Name())
}

// dirResult is used when getting the dirs / files when recrusing in a directory.
type dirResult struct {
	dirs  []fileEntry
	files []fileEntry
}

const (
	RECURSION_LIMIT = 20 // Limit for recursion.
)

var (
	totalFilesCounted int
	totalIgnoredFiles int
)

// ProgramEntry function receives a path string and a config struct, it returns 3 ints in
// order: total amount of files counted, total lines counted and total ignored files. The
// function manages if path that was passed is of a directory or if is from a normal file.
func ProgramEntry(path string, config Config) (int, int, int) {
	if IsDir(path) {
		fileArr := make([]fileEntry, 0, 10)

		recursion := config.Recursion

		dirs := genFileArray(fileArr, getDirs(path), recursion, config)

		return countLinesRecursive(dirs)
	} else {
		return countLinesOfFile(path), -1, -1
	}
}

// countLinesRecursive function count the lines of a file arrays, it uses concorrency, the
// function create workers to count the lines of each directory file concorrently.
func countLinesRecursive(dirs []fileEntry) (int, int, int) {
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

	return totalFilesCounted, totalLines, totalIgnoredFiles
}

// countLinesOfFile function count all the lines of a file passed into it.
func countLinesOfFile(filename string) int {
	if IsDir(filename) {
		totalIgnoredFiles++
		return 0
	}

	fileContent, err := os.ReadFile(filename)
	if err != nil {
		totalIgnoredFiles++
		return 0
	}

	var counter int
	for _, c := range fileContent {
		if c == '\n' {
			counter += 1
		}
	}

	totalFilesCounted++

	return counter
}

// genFileArray function get all the files of a dir and subdir using a slice of fileEntry
// as base, it uses recursion and concorrency with workers to go aggroupate all files into
// a file slice.
func genFileArray(fileArr, dirArr []fileEntry, recLimit int, config Config) []fileEntry {
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
				if strings.HasPrefix(v.Entry.Name(), ".") && !config.NoIgnoreDotFiles {
					continue
				}

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
		fileArr = genFileArray(fileArr, nextDirArr, recLimit-1, config)
	}

	return fileArr
}

// getDirs function returns a slice of fileEntry reading a directory based on a dirPath
// string.
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

// IsDir function returns true if path string is == the path of a directory.
func IsDir(filename string) bool {
	fi, err := os.Stat(filename)
	if err != nil {
		fmt.Println(err)
		return false
	}

	if fi.Mode().IsDir() {
		return true
	}

	return false
}
