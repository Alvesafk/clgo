package core

import (
	"fmt"
	"os"
	"path/filepath"
)

type FileEntry struct {
	Entry os.DirEntry
	Path string
}

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

func CountLinesRecursive(filepath string) {
	dir := getDirs(filepath)

	fmt.Println(genFileArray(dir))
}

func genFileArray(arr []FileEntry) []FileEntry {
	var dirFound int
	for i, v := range arr {
		if v.Entry.IsDir() {
			fullPath := filepath.Join(v.Path, v.Entry.Name())
			arr = append(arr, getDirs(fullPath)...)
			arr = append(arr[:i], arr[i+1:]...)
			dirFound++
		}
	}
	if dirFound > 0 {
		arr = genFileArray(arr)
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
