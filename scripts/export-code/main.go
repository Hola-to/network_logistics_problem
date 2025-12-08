package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// ANSI Colors
var (
	BLUE   = "\033[0;34m"
	CYAN   = "\033[0;36m"
	GREEN  = "\033[0;32m"
	YELLOW = "\033[1;33m"
	GRAY   = "\033[0;90m"
	RED    = "\033[0;31m"
	NC     = "\033[0m"
)

func init() {
	if runtime.GOOS == "windows" {
		if os.Getenv("WT_SESSION") == "" && os.Getenv("TERM_PROGRAM") != "vscode" {
			BLUE, CYAN, GREEN, YELLOW, GRAY, RED, NC = "", "", "", "", "", "", ""
		}
	}
}

// FileInfo holds information about a file
type FileInfo struct {
	Path      string
	RelPath   string
	Extension string
	Lines     int
	Size      int64
}

// CodeExporter exports code to markdown
type CodeExporter struct {
	dirs       []string
	extensions map[string]string // extension -> language for markdown
	exclude    []string
	files      []FileInfo
	stats      struct {
		TotalFiles int
		TotalLines int
		TotalSize  int64
		ByExt      map[string]int
		ByDir      map[string]int
	}
}

// NewCodeExporter creates a new exporter
func NewCodeExporter(dirs []string) *CodeExporter {
	return &CodeExporter{
		dirs: dirs,
		extensions: map[string]string{
			".go":    "go",
			".proto": "protobuf",
			".sql":   "sql",
			".yaml":  "yaml",
			".yml":   "yaml",
			".json":  "json",
			".toml":  "toml",
			".md":    "markdown",
			".sh":    "bash",
			".bash":  "bash",
			".ps1":   "powershell",
			".py":    "python",
			".js":    "javascript",
			".ts":    "typescript",
			".html":  "html",
			".css":   "css",
			".mod":   "go",
			".sum":   "text",
			".txt":   "text",
			".env":   "bash",
		},
		exclude: []string{
			"_test.go",
			".pb.go",
			".pb.gw.go",
			".connect.go",
			"_grpc.pb.go",
			"mock_",
			"mocks/",
			"testdata/",
			"vendor/",
			"node_modules/",
		},
		files: make([]FileInfo, 0),
	}
}

// shouldExclude checks if file should be excluded
func (e *CodeExporter) shouldExclude(path string) bool {
	for _, pattern := range e.exclude {
		if strings.Contains(path, pattern) {
			return true
		}
	}
	return false
}

// getLanguage returns markdown language for extension
func (e *CodeExporter) getLanguage(ext string) string {
	if lang, ok := e.extensions[ext]; ok {
		return lang
	}
	return ""
}

// countLines counts lines in file
func countLines(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lines := 0
	for scanner.Scan() {
		lines++
	}
	return lines, scanner.Err()
}

// collectFiles collects all files from directories
func (e *CodeExporter) collectFiles() error {
	e.stats.ByExt = make(map[string]int)
	e.stats.ByDir = make(map[string]int)

	for _, dir := range e.dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			fmt.Printf("%s  ⚠ Directory not found: %s%s\n", YELLOW, dir, NC)
			continue
		}

		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip errors
			}

			if info.IsDir() {
				return nil
			}

			// Check exclusions
			if e.shouldExclude(path) {
				return nil
			}

			// Check extension
			ext := filepath.Ext(path)
			if _, ok := e.extensions[ext]; !ok {
				return nil
			}

			// Count lines
			lines, err := countLines(path)
			if err != nil {
				return nil
			}

			relPath := path
			fileInfo := FileInfo{
				Path:      path,
				RelPath:   relPath,
				Extension: ext,
				Lines:     lines,
				Size:      info.Size(),
			}

			e.files = append(e.files, fileInfo)
			e.stats.TotalFiles++
			e.stats.TotalLines += lines
			e.stats.TotalSize += info.Size()
			e.stats.ByExt[ext]++

			// Count by top directory
			parts := strings.Split(path, string(os.PathSeparator))
			if len(parts) > 0 {
				e.stats.ByDir[parts[0]]++
			}

			return nil
		})

		if err != nil {
			return err
		}
	}

	// Sort files by path
	sort.Slice(e.files, func(i, j int) bool {
		return e.files[i].RelPath < e.files[j].RelPath
	})

	return nil
}

// Export exports code to markdown file
func (e *CodeExporter) Export(outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Header
	fmt.Fprintf(file, "# Logistics Platform - Source Code\n\n")
	fmt.Fprintf(file, "> Generated: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	// Table of Contents
	fmt.Fprintf(file, "## Table of Contents\n\n")

	currentDir := ""
	for _, f := range e.files {
		dir := filepath.Dir(f.RelPath)
		if dir != currentDir {
			currentDir = dir
			anchor := strings.ReplaceAll(strings.ToLower(dir), "/", "-")
			anchor = strings.ReplaceAll(anchor, ".", "")
			fmt.Fprintf(file, "- [%s](#%s)\n", dir, anchor)
		}
	}
	fmt.Fprintf(file, "\n---\n\n")

	// Statistics
	fmt.Fprintf(file, "## Statistics\n\n")
	fmt.Fprintf(file, "| Metric | Value |\n")
	fmt.Fprintf(file, "|--------|-------|\n")
	fmt.Fprintf(file, "| Total Files | %d |\n", e.stats.TotalFiles)
	fmt.Fprintf(file, "| Total Lines | %d |\n", e.stats.TotalLines)
	fmt.Fprintf(file, "| Total Size | %.2f KB |\n", float64(e.stats.TotalSize)/1024)
	fmt.Fprintf(file, "\n")

	fmt.Fprintf(file, "### By Extension\n\n")
	fmt.Fprintf(file, "| Extension | Files |\n")
	fmt.Fprintf(file, "|-----------|-------|\n")
	exts := make([]string, 0, len(e.stats.ByExt))
	for ext := range e.stats.ByExt {
		exts = append(exts, ext)
	}
	sort.Strings(exts)
	for _, ext := range exts {
		fmt.Fprintf(file, "| %s | %d |\n", ext, e.stats.ByExt[ext])
	}
	fmt.Fprintf(file, "\n")

	fmt.Fprintf(file, "### By Directory\n\n")
	fmt.Fprintf(file, "| Directory | Files |\n")
	fmt.Fprintf(file, "|-----------|-------|\n")
	dirs := make([]string, 0, len(e.stats.ByDir))
	for dir := range e.stats.ByDir {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)
	for _, dir := range dirs {
		fmt.Fprintf(file, "| %s | %d |\n", dir, e.stats.ByDir[dir])
	}
	fmt.Fprintf(file, "\n---\n\n")

	// Files content
	fmt.Fprintf(file, "## Source Files\n\n")

	currentDir = ""
	for _, f := range e.files {
		dir := filepath.Dir(f.RelPath)
		if dir != currentDir {
			currentDir = dir
			fmt.Fprintf(file, "### %s\n\n", dir)
		}

		// File header
		fmt.Fprintf(file, "#### `%s`\n\n", filepath.Base(f.RelPath))
		fmt.Fprintf(file, "> Path: `%s` | Lines: %d\n\n", f.RelPath, f.Lines)

		// Read file content
		content, err := os.ReadFile(f.Path)
		if err != nil {
			fmt.Fprintf(file, "```\nError reading file: %v\n```\n\n", err)
			continue
		}

		// Write code block
		lang := e.getLanguage(f.Extension)
		fmt.Fprintf(file, "```%s\n", lang)
		fmt.Fprintf(file, "%s", string(content))
		if !strings.HasSuffix(string(content), "\n") {
			fmt.Fprintf(file, "\n")
		}
		fmt.Fprintf(file, "```\n\n")
	}

	return nil
}

func printHeader() {
	fmt.Printf("%s╔═══════════════════════════════════════════════════════════════════╗%s\n", CYAN, NC)
	fmt.Printf("%s║       Code Exporter                                               ║%s\n", CYAN, NC)
	fmt.Printf("%s╚═══════════════════════════════════════════════════════════════════╝%s\n", CYAN, NC)
	fmt.Println()
}

func main() {
	// Flags
	dirsFlag := flag.String("dirs", "api,pkg,services,migrations", "Comma-separated directories to export")
	output := flag.String("output", "logistics-code.md", "Output file path")
	excludeFlag := flag.String("exclude", "_test.go,.pb.go,_grpc.pb.go,.connect.go,mock_,mocks/,testdata/", "Comma-separated patterns to exclude")
	includeTests := flag.Bool("include-tests", false, "Include test files")
	includeGenerated := flag.Bool("include-generated", false, "Include generated files (.pb.go, etc.)")
	flag.Parse()

	printHeader()

	// Parse directories
	dirs := strings.Split(*dirsFlag, ",")
	for i := range dirs {
		dirs[i] = strings.TrimSpace(dirs[i])
	}

	fmt.Printf("%s[1/3] Initializing...%s\n", BLUE, NC)
	fmt.Printf("  Directories: %s%v%s\n", YELLOW, dirs, NC)
	fmt.Printf("  Output: %s%s%s\n", YELLOW, *output, NC)

	exporter := NewCodeExporter(dirs)

	// Handle exclusion flags
	if !*includeTests {
		// Already excluded by default
	} else {
		// Remove test exclusions
		var newExclude []string
		for _, e := range exporter.exclude {
			if !strings.Contains(e, "_test.go") && !strings.Contains(e, "testdata") {
				newExclude = append(newExclude, e)
			}
		}
		exporter.exclude = newExclude
	}

	if *includeGenerated {
		var newExclude []string
		for _, e := range exporter.exclude {
			if !strings.Contains(e, ".pb.go") && !strings.Contains(e, ".connect.go") {
				newExclude = append(newExclude, e)
			}
		}
		exporter.exclude = newExclude
	}

	// Add custom exclusions
	if *excludeFlag != "" {
		for _, pattern := range strings.Split(*excludeFlag, ",") {
			pattern = strings.TrimSpace(pattern)
			if pattern != "" {
				exporter.exclude = append(exporter.exclude, pattern)
			}
		}
	}

	fmt.Printf("\n%s[2/3] Collecting files...%s\n", BLUE, NC)
	if err := exporter.collectFiles(); err != nil {
		fmt.Fprintf(os.Stderr, "%sError: %v%s\n", RED, err, NC)
		os.Exit(1)
	}

	fmt.Printf("  Found %s%d%s files (%s%d%s lines)\n",
		GREEN, exporter.stats.TotalFiles, NC,
		YELLOW, exporter.stats.TotalLines, NC)

	fmt.Printf("\n%s[3/3] Exporting to %s...%s\n", BLUE, *output, NC)
	if err := exporter.Export(*output); err != nil {
		fmt.Fprintf(os.Stderr, "%sError: %v%s\n", RED, err, NC)
		os.Exit(1)
	}

	// Summary
	fmt.Printf("\n%s=== Export Complete ===%s\n", GREEN, NC)
	fmt.Printf("  Output: %s%s%s\n", CYAN, *output, NC)
	fmt.Printf("  Files: %s%d%s\n", GREEN, exporter.stats.TotalFiles, NC)
	fmt.Printf("  Lines: %s%d%s\n", GREEN, exporter.stats.TotalLines, NC)
	fmt.Printf("  Size: %s%.2f KB%s\n", GREEN, float64(exporter.stats.TotalSize)/1024, NC)

	fmt.Println()
	fmt.Printf("%s╔═══════════════════════════════════════════════════════════════════╗%s\n", CYAN, NC)
	fmt.Printf("%s║       Done!                                                       ║%s\n", CYAN, NC)
	fmt.Printf("%s╚═══════════════════════════════════════════════════════════════════╝%s\n", CYAN, NC)
}
