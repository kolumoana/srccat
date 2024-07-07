package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/gobwas/glob"
	"github.com/urfave/cli/v2"
)

const (
	maxFileSize = 1 * 1024 * 1024 // 1MB
)

type FileContent struct {
	Path    string `json:"path"`
	Content string `json:"content,omitempty"`
}

var customExcludePatterns []glob.Glob

func main() {
	app := &cli.App{
		Name:  "srccat",
		Usage: "List and display contents of source code files in a directory, respecting .gitignore and excluding unnecessary files",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "dir",
				Aliases:  []string{"d"},
				Usage:    "Directory to process",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"f"},
				Usage:   "Output format (text or json)",
				Value:   "text",
			},
			&cli.BoolFlag{
				Name:    "list",
				Aliases: []string{"l"},
				Usage:   "Output only the list of file names",
			},
			&cli.StringSliceFlag{
				Name:    "exclude",
				Aliases: []string{"e"},
				Usage:   "Custom exclude patterns (e.g. '*.css', '*.md')",
			},
		},
		Action: func(c *cli.Context) error {
			directory := c.String("dir")
			format := c.String("format")
			listOnly := c.Bool("list")
			excludePatterns := c.StringSlice("exclude")

			if _, err := os.Stat(directory); os.IsNotExist(err) {
				return fmt.Errorf("error: specified directory does not exist")
			}

			if !listOnly && format != "text" && format != "json" {
				return fmt.Errorf("error: invalid format specified. Use 'text' or 'json'")
			}

			if listOnly {
				format = "list"
			}

			for _, pattern := range excludePatterns {
				g, err := glob.Compile(pattern)
				if err != nil {
					return fmt.Errorf("invalid exclude pattern '%s': %v", pattern, err)
				}
				customExcludePatterns = append(customExcludePatterns, g)
			}

			return listFiles(directory, format)
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func listFiles(baseDir, format string) error {
	repo, err := git.PlainOpen(baseDir)
	if err != nil && err != git.ErrRepositoryNotExists {
		return fmt.Errorf("failed to open git repository: %w", err)
	}

	var ignorePatterns []gitignore.Pattern
	if repo != nil {
		wt, err := repo.Worktree()
		if err != nil {
			return fmt.Errorf("failed to get worktree: %w", err)
		}
		ignorePatterns = wt.Excludes
	}

	matcher := gitignore.NewMatcher(ignorePatterns)

	var files []FileContent
	var mu sync.Mutex
	var wg sync.WaitGroup

	progressChan := make(chan string)
	doneChan := make(chan bool)

	go showProgress(progressChan, doneChan)

	err = filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relativePath, err := filepath.Rel(baseDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		if shouldExclude(info.Name(), relativePath, matcher) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if !info.IsDir() {
			wg.Add(1)
			go func() {
				defer wg.Done()
				processFile(path, relativePath, &files, &mu, progressChan, format != "list")
			}()
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("error walking the path %s: %w", baseDir, err)
	}

	wg.Wait()
	close(progressChan)
	<-doneChan

	switch format {
	case "json":
		return outputJSON(files)
	case "list":
		return outputList(files)
	default:
		return outputText(files)
	}
}

func processFile(path, relativePath string, files *[]FileContent, mu *sync.Mutex, progressChan chan<- string, readContent bool) {
	info, err := os.Stat(path)
	if err != nil {
		log.Printf("Error getting file info for %s: %v", path, err)
		return
	}

	if info.Size() > maxFileSize {
		log.Printf("Skipping large file: %s (size: %d bytes)", path, info.Size())
		return
	}

	fileContent := FileContent{Path: relativePath}

	if readContent {
		content, err := os.ReadFile(path)
		if err != nil {
			log.Printf("Failed to read file %s: %v", path, err)
			return
		}

		if !isBinary(content) {
			fileContent.Content = string(content)
		}
	}

	mu.Lock()
	*files = append(*files, fileContent)
	mu.Unlock()

	progressChan <- relativePath
}

func shouldExclude(name, relativePath string, matcher gitignore.Matcher) bool {
	excludedDirs := []string{
		".git", "node_modules", "build", "dist", "out", ".cache", ".tmp",
		".vscode", ".idea", ".next", "public", ".terraform",
	}
	for _, dir := range excludedDirs {
		if name == dir || strings.HasPrefix(relativePath, dir+string(os.PathSeparator)) {
			return true
		}
	}

	excludedPatterns := []string{
		".json", ".log", ".bak", "~", ".DS_Store",
		"package-lock.json", "yarn.lock", ".d.ts", "config.mjs",
		".lock.hcl", ".ico", ".tfstate",
		".backup", ".pptx", ".ppt", ".doc", ".docx", ".xls", ".xlsx", ".mod", ".sum",
	}
	for _, pattern := range excludedPatterns {
		if strings.HasSuffix(name, pattern) {
			return true
		}
	}

	// .env ファイルの処理
	if strings.HasPrefix(name, ".env") {
		return true
	}

	// 特定のファイル名を除外
	if name == ".gitignore" {
		return true
	}

	// カスタム除外パターンのチェック
	for _, pattern := range customExcludePatterns {
		if pattern.Match(relativePath) {
			return true
		}
	}

	return matcher.Match([]string{relativePath}, false)
}

func isBinary(content []byte) bool {
	if len(content) > 1024 {
		content = content[:1024]
	}
	for _, b := range content {
		if b == 0 {
			return true
		}
	}
	return false
}

func outputJSON(files []FileContent) error {
	return json.NewEncoder(os.Stdout).Encode(files)
}

func outputText(files []FileContent) error {
	for _, file := range files {
		fmt.Printf("\n```%s\n%s\n```\n", file.Path, file.Content)
	}
	return nil
}

func outputList(files []FileContent) error {
	for _, file := range files {
		fmt.Printf("- %s\n", file.Path)
	}
	return nil
}

func showProgress(progressChan <-chan string, doneChan chan<- bool) {
	start := time.Now()
	count := 0

	for range progressChan {
		count++
		if count%100 == 0 {
			elapsed := time.Since(start)
			fmt.Fprintf(os.Stderr, "\rProcessed %d files in %v", count, elapsed.Round(time.Second))
		}
	}

	elapsed := time.Since(start)
	fmt.Fprintf(os.Stderr, "\rProcessed %d files in %v\n", count, elapsed.Round(time.Second))
	doneChan <- true
}
