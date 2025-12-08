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
)

// ANSI Colors
var (
	BLUE   = "\033[0;34m"
	CYAN   = "\033[0;36m"
	GREEN  = "\033[0;32m"
	YELLOW = "\033[1;33m"
	GRAY   = "\033[0;90m"
	BOLD   = "\033[1m"
	NC     = "\033[0m"
)

func init() {
	if runtime.GOOS == "windows" {
		if os.Getenv("WT_SESSION") == "" && os.Getenv("TERM_PROGRAM") != "vscode" {
			BLUE, CYAN, GREEN, YELLOW, GRAY, BOLD, NC = "", "", "", "", "", "", ""
		}
	}
}

// FileStats holds statistics for a file type
type FileStats struct {
	Extension string
	Files     int
	Lines     int
	Blank     int
	Comment   int
	Code      int
}

// DirStats holds statistics for a directory
type DirStats struct {
	Name  string
	Files int
	Lines int
	Code  int
}

// LOCCounter counts lines of code
type LOCCounter struct {
	root        string
	excludeDirs []string
	extensions  map[string]bool
	stats       map[string]*FileStats
	dirStats    map[string]*DirStats
	totalFiles  int
	totalLines  int
	totalBlank  int
	totalCode   int
}

// NewLOCCounter creates a new counter
func NewLOCCounter(root string) *LOCCounter {
	return &LOCCounter{
		root: root,
		excludeDirs: []string{
			".git",
			"vendor",
			"node_modules",
			".idea",
			".vscode",
			".zed",
		},
		extensions: map[string]bool{
			".go":    true,
			".proto": true,
			".sql":   true,
			".yaml":  true,
			".yml":   true,
			".json":  true,
			".toml":  true,
			".md":    true,
			".sh":    true,
			".bash":  true,
			".ps1":   true,
			".py":    true,
			".js":    true,
			".ts":    true,
			".html":  true,
			".css":   true,
			".mod":   true,
			".sum":   true,
			".txt":   true,
		},
		stats:    make(map[string]*FileStats),
		dirStats: make(map[string]*DirStats),
	}
}

// shouldExclude checks if path should be excluded
func (c *LOCCounter) shouldExclude(path string) bool {
	for _, dir := range c.excludeDirs {
		if strings.Contains(path, string(os.PathSeparator)+dir+string(os.PathSeparator)) ||
			strings.HasSuffix(path, string(os.PathSeparator)+dir) ||
			strings.HasPrefix(path, dir+string(os.PathSeparator)) ||
			path == dir {
			return true
		}
	}
	return false
}

// countFile counts lines in a single file
func (c *LOCCounter) countFile(path string) (total, blank, code int, err error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0, 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		total++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			blank++
		} else {
			code++
		}
	}

	return total, blank, code, scanner.Err()
}

// Count counts all lines
func (c *LOCCounter) Count() error {
	return filepath.Walk(c.root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Skip excluded directories
		if info.IsDir() {
			if c.shouldExclude(path) {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip excluded paths
		if c.shouldExclude(path) {
			return nil
		}

		// Check extension
		ext := filepath.Ext(path)
		if !c.extensions[ext] {
			return nil
		}

		// Count lines
		total, blank, code, err := c.countFile(path)
		if err != nil {
			return nil
		}

		// Update extension stats
		if _, ok := c.stats[ext]; !ok {
			c.stats[ext] = &FileStats{Extension: ext}
		}
		c.stats[ext].Files++
		c.stats[ext].Lines += total
		c.stats[ext].Blank += blank
		c.stats[ext].Code += code

		// Update directory stats
		parts := strings.Split(path, string(os.PathSeparator))
		topDir := parts[0]
		if topDir == "." && len(parts) > 1 {
			topDir = parts[1]
		}
		if _, ok := c.dirStats[topDir]; !ok {
			c.dirStats[topDir] = &DirStats{Name: topDir}
		}
		c.dirStats[topDir].Files++
		c.dirStats[topDir].Lines += total
		c.dirStats[topDir].Code += code

		// Update totals
		c.totalFiles++
		c.totalLines += total
		c.totalBlank += blank
		c.totalCode += code

		return nil
	})
}

// PrintStats prints statistics
func (c *LOCCounter) PrintStats(detailed bool) {
	fmt.Printf("\n%s╔═══════════════════════════════════════════════════════════════════╗%s\n", CYAN, NC)
	fmt.Printf("%s║       Lines of Code Counter                                       ║%s\n", CYAN, NC)
	fmt.Printf("%s╚═══════════════════════════════════════════════════════════════════╝%s\n", CYAN, NC)
	fmt.Println()

	if detailed {
		// By extension
		fmt.Printf("%s=== By Extension ===%s\n\n", GREEN, NC)
		fmt.Printf("%-12s %10s %12s %12s %12s\n", "Extension", "Files", "Lines", "Blank", "Code")
		fmt.Println(strings.Repeat("─", 60))

		exts := make([]string, 0, len(c.stats))
		for ext := range c.stats {
			exts = append(exts, ext)
		}
		sort.Slice(exts, func(i, j int) bool {
			return c.stats[exts[i]].Code > c.stats[exts[j]].Code
		})

		for _, ext := range exts {
			s := c.stats[ext]
			fmt.Printf("%-12s %10d %12d %12d %12d\n", ext, s.Files, s.Lines, s.Blank, s.Code)
		}

		fmt.Println(strings.Repeat("─", 60))
		fmt.Printf("%s%-12s %10d %12d %12d %12d%s\n", BOLD, "TOTAL", c.totalFiles, c.totalLines, c.totalBlank, c.totalCode, NC)

		// By directory
		fmt.Printf("\n%s=== By Directory ===%s\n\n", GREEN, NC)
		fmt.Printf("%-20s %10s %12s %12s\n", "Directory", "Files", "Lines", "Code")
		fmt.Println(strings.Repeat("─", 56))

		dirs := make([]string, 0, len(c.dirStats))
		for dir := range c.dirStats {
			dirs = append(dirs, dir)
		}
		sort.Slice(dirs, func(i, j int) bool {
			return c.dirStats[dirs[i]].Code > c.dirStats[dirs[j]].Code
		})

		for _, dir := range dirs {
			s := c.dirStats[dir]
			fmt.Printf("%-20s %10d %12d %12d\n", dir, s.Files, s.Lines, s.Code)
		}

		fmt.Println(strings.Repeat("─", 56))
	}

	// Summary
	fmt.Printf("\n%s=== Summary ===%s\n\n", GREEN, NC)
	fmt.Printf("  Total Files: %s%d%s\n", YELLOW, c.totalFiles, NC)
	fmt.Printf("  Total Lines: %s%d%s\n", YELLOW, c.totalLines, NC)
	fmt.Printf("  Blank Lines: %s%d%s\n", GRAY, c.totalBlank, NC)
	fmt.Printf("  Code Lines:  %s%s%d%s\n", BOLD, GREEN, c.totalCode, NC)

	// Percentages
	if c.totalLines > 0 {
		codePercent := float64(c.totalCode) / float64(c.totalLines) * 100
		blankPercent := float64(c.totalBlank) / float64(c.totalLines) * 100
		fmt.Printf("\n  Code:  %s%.1f%%%s\n", GREEN, codePercent, NC)
		fmt.Printf("  Blank: %s%.1f%%%s\n", GRAY, blankPercent, NC)
	}

	fmt.Println()
}

// PrintSimple prints just the total
func (c *LOCCounter) PrintSimple() {
	fmt.Println(c.totalCode)
}

func main() {
	rootPath := flag.String("path", ".", "Root directory path")
	detailed := flag.Bool("detailed", true, "Show detailed statistics")
	simple := flag.Bool("simple", false, "Print only total lines of code")
	excludeFlag := flag.String("exclude", ".git,vendor,node_modules,.idea,.vscode,.zed", "Comma-separated directories to exclude")
	flag.Parse()

	if flag.NArg() > 0 {
		*rootPath = flag.Arg(0)
	}

	counter := NewLOCCounter(*rootPath)

	// Parse exclusions
	if *excludeFlag != "" {
		counter.excludeDirs = strings.Split(*excludeFlag, ",")
		for i := range counter.excludeDirs {
			counter.excludeDirs[i] = strings.TrimSpace(counter.excludeDirs[i])
		}
	}

	if err := counter.Count(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if *simple {
		counter.PrintSimple()
	} else {
		counter.PrintStats(*detailed)
	}
}
