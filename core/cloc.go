/*
Copyright (c) 2026 Alvesafk. All Rights Reserved.

Core package has the business logic of clgo.
*/
package core

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
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

// fileStats struct represent stats from a file.
type fileStats struct {
	Language     string
	CodeLines    int
	CommentLines int
	BlankLines   int
}

// LanguageStats is the main struct passed to main, it has the total of files, code lines,
// comment lines and blank lines.
type LanguageStats struct {
	Files        int
	CodeLines    int
	CommentLines int
	BlankLines   int
}

// comment markers represents how a comment is defined in a language, it's one liner and
// multi line variants.
type commentMarkers struct {
	Line  []string
	Open  string
	Close string
}

const (
	RECURSION_LIMIT = 20 // Limit for recursion.
)

var (
	totalFilesCounted int
	totalSkippedFiles int
)

// ProgramEntry function receives a path string and a config struct, it returns a map and
// two ints, the map is LanguageStats map with the stats of all parsed files, the two ints
// are: total files counted and total skipped files.
func ProgramEntry(path string, config Config) (map[string]LanguageStats, int, int) {
	if IsDir(path) {
		fileArr := make([]fileEntry, 0, 10)

		recursion := config.Recursion

		dirs := genFileArray(fileArr, getDirs(path), recursion, config)

		return countLinesRecursive(dirs)
	}

	languages := make(map[string]LanguageStats)

	stats, ok := countLinesOfFile(path)
	if ok {
		languages[stats.Language] = LanguageStats{
			Files:        1,
			CodeLines:    stats.CodeLines,
			CommentLines: stats.CommentLines,
			BlankLines:   stats.BlankLines,
		}
	}

	return languages, totalFilesCounted, totalSkippedFiles
}

// countLinesRecursive function count the lines of a file slice, it uses concorrency, the
// function create workers to count the lines of each directory file concorrently.
func countLinesRecursive(dirs []fileEntry) (map[string]LanguageStats, int, int) {
	jobs := make(chan fileEntry, len(dirs))
	results := make(chan fileStats, len(dirs))

	numWorkers := runtime.NumCPU() / 2
	var wg sync.WaitGroup

	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for v := range jobs {
				if stats, ok := countLinesOfFile(v.fullpath()); ok {
					results <- stats
				}
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

	languages := make(map[string]LanguageStats)
	for r := range results {
		lang := languages[r.Language]
		lang.Files++
		lang.CodeLines += r.CodeLines
		lang.CommentLines += r.CommentLines
		lang.BlankLines += r.BlankLines
		languages[r.Language] = lang
	}

	return languages, totalFilesCounted, totalSkippedFiles
}

// countLinesOfFile function parse a file couting it's code, blank and comment lines.
func countLinesOfFile(filename string) (fileStats, bool) {
	if IsDir(filename) {
		return fileStats{}, false
	}

	if slices.Contains(filenameToIgnore, filepath.Base(filename)) {
		return fileStats{}, false
	}

	file, err := os.Open(filename)
	if err != nil {
		totalSkippedFiles++
		return fileStats{}, false
	}
	defer file.Close()

	language, ignore := languageFromExt(filename)
	if ignore {
		return fileStats{}, false
	}

	markers, hasSyntax := commentSyntax[language]

	stats := fileStats{Language: language}
	var insideBlock bool

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			stats.BlankLines++
			continue
		}

		if hasSyntax {
			if insideBlock {
				stats.CommentLines++
				if markers.Close != "" && strings.Contains(trimmed, markers.Close) {
					insideBlock = false
				}
				continue
			}

			if markers.Open != "" && strings.HasPrefix(trimmed, markers.Open) {
				stats.CommentLines++
				if !strings.Contains(trimmed, markers.Close) {
					insideBlock = true
				}
				continue
			}

			if len(markers.Line) > 0 && checkCommentPrefix(trimmed, markers) {
				stats.CommentLines++
				continue
			}
		}

		stats.CodeLines++
	}

	if err := scanner.Err(); err != nil {
		totalSkippedFiles++
		return fileStats{}, false
	}

	totalFilesCounted++

	return stats, true
}

// Returns name of the lang after comparing it to the suffix map, if not found returns
// "Unknown"
func languageFromExt(filename string) (string, bool) {
	baseName := filepath.Base(filename)

	ext := filepath.Ext(filename)
	if !strings.Contains(ext, ".") {
		if file, ok := filenameException[baseName]; ok {
			return file, false
		}

		return "Unknown", false
	}

	if slices.Contains(extToIgnore, ext) {
		return "", true
	}

	if lang, ok := extToLanguage[ext]; ok {
		return lang, false
	}

	return "Unknown", false
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
				if isBin, _ := isBinary(v.fullpath()); strings.HasPrefix(v.Entry.Name(), ".") && !config.NoIgnoreDotFiles || isBin {
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
func IsDir(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		fmt.Println(err)
		return false
	}

	if fi.Mode().IsDir() {
		return true
	}

	return false
}

// isBinary function returns true if path string is the path of a binary file,
// the function checks for a "0x00" byte insede the first 8000 bytes, it's how
// git does this.
func isBinary(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	buf := make([]byte, 8000)
	n, err := f.Read(buf)
	if err != nil && n == 0 {
		return false, err
	}

	return bytes.IndexByte(buf[:n], 0) != -1, nil
}

func checkCommentPrefix(trimmed string, markers commentMarkers) bool {
	for _, v := range markers.Line {
		if strings.HasPrefix(trimmed, v) {
			return true
		}
	}

	return false
}
