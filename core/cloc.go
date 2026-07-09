package core

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

type Config struct {
	NoRecursion      bool
	NoStats          bool
	NoIgnoreDotFiles bool
}

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
	totalIgnoredFiles int
)

func ProgramEntry(path string, config Config) (int, int ,int) {
	if IsDir(path) {
		fileArr := make([]fileEntry, 0, 10)

		recursion := RECURSION_LIMIT
		if config.NoRecursion {
			recursion = 0
		}

		dirs := genFileArray(fileArr, getDirs(path), recursion, config)

		return countLinesRecursive(dirs)
	} else {
		return countLinesOfFile(path), -1, -1
	}
}

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
