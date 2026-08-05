/*
SPDX-License-Identifier: GPL-3.0-only
Copyright (c) 2026 Alvesafk.
*/
package cloc

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type directoryJob struct {
	path  string
	depth int
	rules *ignore.Matcher
}

func countDirectorySequential(ctx context.Context, root string, config Config, matcher *ignore.Matcher, result *Result, metrics *Metrics) error {
	queue := []directoryJob{{path: root, depth: 0, rules: matcher}}
	for len(queue) > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}

		job := queue[0]
		queue = queue[1:]

		entries, err := os.ReadDir(job.path)
		if err != nil {
			if job.depth == 0 {
				fmt.Errorf("read directory %q: %w", job.path, err)
			}

			addDirectoryWarning(result, metrics, job.path, err)
			continue
		}

		rules := job.rules
		if config.UseGitIgnore {
			rules, err = rules.WithDirectoryRules(job.path)
			if err != nil {
				addDirectoryWarning(result, metrics, filepath.Join(job.path, ".gitignore"), err)
				rules = job.rules
			}
		}

		for _, entry := range entries {
			fullPath := filepath.Join(job.path, entry.Name())
			relative, relErr := filepath.Rel(root, fullPath)
			if relErr != nil {
				addDirectoryWarning(result, metrics, fullPath, relErr)
				continue
			}

			isDir := entry.IsDir()
			if shouldIgnorePath(relative, entry.Name(), isDir, config, rules) {
				if !isDir {
					metrics.filesIgnored.Add(1)
				}

				continue
			}

			if isDir {
				if job.depth < config.RecursionLimit {
					queue = append(queue, directoryJob{path: fullPath, depth: job.depth + 1, rules: rules})
				}

				continue
			}

			if !entry.Type().IsRegular() {
				continue
			}

			metrics.filesDiscovered.Add(1)
			applyOutcome(result, processFile(ctx, fullPath, config), metrics)
		}

	}

	return nil
}

func countDirectoryPipeline(ctx context.Context, root string, config Config, matcher *ignore.Matcher, result *Result, metrics *Metrics) error {
	jobs := make(chan string, config.Workers*2)
	outcomes := make(chan fileOutcome, config.Workers*2)
	walkErrors := make(chan error, 1)
	accumulator := &resultAccumulator{result: result, metrics: metrics}

	var workers sync.WaitGroup
	for i := 0; i < config.Workers; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for path := range jobs {
				outcome := processFile(ctx, path, config)
				select {
				case outcomes <- outcome:
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	go func() {
		defer close(jobs)
		walkErrors <- walkAndQueue(ctx, root, config, matcher, jobs, accumulator, metrics)
		close(walkErrors)
	}()

	go func() {
		workers.Wait()
		close(outcomes)
	}()

	for outcome := range outcomes {
		accumulator.addOutcome(outcome)
	}

	if err := <-walkErrors; err != nil {
		return err
	}

	return ctx.Err()
}

func walkAndQueue(ctx context.Context, root string, config Config, matcher *ignore.Matcher, jobs chan<- string, accumulator *resultAccumulator, metrics *Metrics) error {
	queue := []directoryJob{{path: root, depth: 0, rules: matcher}}

	for len(queue) > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}

		job := queue[0]
		queue = queue[1:]
		entries, err := os.ReadDir(job.path)
		if err != nil {
			if job.depth == 0 {
				return fmt.Errorf("read directory %q: %w", job.path, err)
			}

			accumulator.addDirectoryWarning(job.path, err)
			continue
		}

		rules := job.rules
		if config.UseGitIgnore {
			rules, err = rules.WithDirectoryRules(job.path)
			if err != nil {
				accumulator.addDirectoryWarning(filepath.Join(job.path, ".gitignore"), err)
				rules = job.rules
			}
		}

		for _, entry := range entries {
			fullPath := filepath.Join(job.path, entry.Name())
			relative, relErr := filepath.Rel(root, fullPath)
			if relErr != nil {
				accumulator.addDirectoryWarning(fullPath, relErr)
				continue
			}

			isDir := entry.IsDir()
			if shouldIgnorePath(relative, entry.Name(), isDir, config, rules) {
				if !isDir {
					metrics.filesIgnored.Add(1)
				}

				continue
			}

			if isDir {
				if job.depth < config.RecursionLimit {
					queue = append(queue, directoryJob{path: fullPath, depth: job.depth + 1, rules: rules})
				}

				continue
			}

			if !entry.Type().IsRegular() {
				continue
			}

			metrics.filesDiscovered.Add(1)
			select {
			case jobs <- fullPath:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return nil
}

func shouldIgnorePath(relative, name string, isDir bool, config Config, matcher *ignore.Matcher) bool {
	if langs.IgnoreFilename(name) {
		return true
	}

	if strings.HasPrefix(name, ".") && !config.IncludeHidden {
		return true
	}

	if isDir && matchesDirectory(relative, name, config.ExcludeDirs) {
		return true
	}

	if matcher != nil && matcher.Match(relative, isDir) {
		return true
	}

	if !isDir {
		if extensionIgnored(relative, config.IgnoreExtensions) {
			return true
		}

		if len(config.IncludePatterns) > 0 && !ignore.MatchAny(config.IncludePatterns, relative) {
			return true
		}

		if ignore.MatchAny(config.ExcludePatterns, relative) {
			return true
		}
	}

	return false
}

func matchesDirectory(relative, name string, patterns []string) bool {
	for _, pattern := range patterns {
		pattern = filepath.ToSlash(strings.TrimSpace(pattern))
		if pattern == "" {
			continue
		}

		if pattern == name || ignore.Match(pattern, filepath.ToSlash(relative)) {
			return true
		}
	}

	return false
}
