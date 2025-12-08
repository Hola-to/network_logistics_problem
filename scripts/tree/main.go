package main

import (
	"flag"
	"fmt"
	"io/fs"
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

// Stats holds directory statistics
type Stats struct {
	Dirs  int
	Files int
}

// TreePrinter prints directory tree
type TreePrinter struct {
	root        string
	excludeDirs map[string]bool
	showHidden  bool
	maxDepth    int
	dirsOnly    bool
	stats       Stats
	output      *os.File
}

// NewTreePrinter creates a new TreePrinter
func NewTreePrinter(root string) *TreePrinter {
	return &TreePrinter{
		root: root,
		excludeDirs: map[string]bool{
			".git": true, // Exclude .git contents but show the directory
		},
		showHidden: true,
		maxDepth:   -1, // No limit
		dirsOnly:   false,
		output:     os.Stdout,
	}
}

// printEntry prints a single tree entry
func (t *TreePrinter) printEntry(name string, prefix string, isLast bool, isDir bool) {
	connector := "├── "
	if isLast {
		connector = "└── "
	}

	color := ""
	suffix := ""
	if isDir {
		color = BLUE
		suffix = "/"
		t.stats.Dirs++
	} else {
		t.stats.Files++
	}

	fmt.Fprintf(t.output, "%s%s%s%s%s%s\n", prefix, connector, color, name, suffix, NC)
}

// shouldExcludeContents checks if directory contents should be excluded
func (t *TreePrinter) shouldExcludeContents(name string) bool {
	return t.excludeDirs[name]
}

// shouldSkip checks if entry should be skipped entirely
func (t *TreePrinter) shouldSkip(name string) bool {
	if !t.showHidden && strings.HasPrefix(name, ".") {
		return true
	}
	return false
}

// walkDir walks directory and prints tree
func (t *TreePrinter) walkDir(path string, prefix string, depth int) error {
	if t.maxDepth >= 0 && depth > t.maxDepth {
		return nil
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	// Filter and sort entries
	var filtered []fs.DirEntry
	for _, entry := range entries {
		if !t.shouldSkip(entry.Name()) {
			if !t.dirsOnly || entry.IsDir() {
				filtered = append(filtered, entry)
			}
		}
	}

	// Sort: directories first, then alphabetically
	sort.Slice(filtered, func(i, j int) bool {
		iDir := filtered[i].IsDir()
		jDir := filtered[j].IsDir()
		if iDir != jDir {
			return iDir
		}
		return strings.ToLower(filtered[i].Name()) < strings.ToLower(filtered[j].Name())
	})

	for i, entry := range filtered {
		isLast := i == len(filtered)-1
		name := entry.Name()

		t.printEntry(name, prefix, isLast, entry.IsDir())

		if entry.IsDir() {
			// Check if we should skip contents
			if t.shouldExcludeContents(name) {
				continue
			}

			newPrefix := prefix
			if isLast {
				newPrefix += "    "
			} else {
				newPrefix += "│   "
			}

			subPath := filepath.Join(path, name)
			if err := t.walkDir(subPath, newPrefix, depth+1); err != nil {
				// Continue on error
				continue
			}
		}
	}

	return nil
}

// Print prints the directory tree
func (t *TreePrinter) Print() error {
	// Print root
	absPath, _ := filepath.Abs(t.root)
	rootName := filepath.Base(absPath)
	if t.root == "." {
		rootName, _ = os.Getwd()
		rootName = filepath.Base(rootName)
	}

	fmt.Fprintf(t.output, "%s%s%s/\n", BOLD+BLUE, rootName, NC)

	if err := t.walkDir(t.root, "", 0); err != nil {
		return err
	}

	return nil
}

// PrintStats prints statistics
func (t *TreePrinter) PrintStats() {
	fmt.Fprintf(t.output, "\n%s%d directories, %d files%s\n", GRAY, t.stats.Dirs, t.stats.Files, NC)
}

func printHeader(output *os.File) {
	fmt.Fprintln(output)
	fmt.Fprintf(output, "%s╔═══════════════════════════════════════════════════════════════════╗%s\n", CYAN, NC)
	fmt.Fprintf(output, "%s║       Directory Tree                                              ║%s\n", CYAN, NC)
	fmt.Fprintf(output, "%s╚═══════════════════════════════════════════════════════════════════╝%s\n", CYAN, NC)
	fmt.Fprintln(output)
}

func main() {
	// Flags
	rootPath := flag.String("path", ".", "Root directory path")
	maxDepth := flag.Int("depth", -1, "Maximum depth (-1 for unlimited)")
	dirsOnly := flag.Bool("dirs", false, "Show directories only")
	noHidden := flag.Bool("no-hidden", false, "Hide hidden files/directories")
	outputFile := flag.String("output", "", "Output file (default: stdout)")
	noHeader := flag.Bool("no-header", false, "Don't print header")
	noStats := flag.Bool("no-stats", false, "Don't print statistics")
	exclude := flag.String("exclude", ".git", "Comma-separated directories to exclude contents")
	flag.Parse()

	// Handle positional argument
	if flag.NArg() > 0 {
		*rootPath = flag.Arg(0)
	}

	// Create printer
	printer := NewTreePrinter(*rootPath)
	printer.maxDepth = *maxDepth
	printer.dirsOnly = *dirsOnly
	printer.showHidden = !*noHidden

	// Parse exclude list
	if *exclude != "" {
		printer.excludeDirs = make(map[string]bool)
		for _, dir := range strings.Split(*exclude, ",") {
			printer.excludeDirs[strings.TrimSpace(dir)] = true
		}
	}

	// Output file
	if *outputFile != "" {
		f, err := os.Create(*outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		printer.output = f
		// Disable colors for file output
		BLUE, CYAN, GREEN, YELLOW, GRAY, BOLD, NC = "", "", "", "", "", "", ""
	}

	if !*noHeader {
		printHeader(printer.output)
	}

	if err := printer.Print(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if !*noStats {
		printer.PrintStats()
	}
}
