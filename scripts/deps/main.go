package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// ANSI Colors
var (
	RED    = "\033[0;31m"
	GREEN  = "\033[0;32m"
	YELLOW = "\033[1;33m"
	BLUE   = "\033[0;34m"
	CYAN   = "\033[0;36m"
	GRAY   = "\033[0;90m"
	BOLD   = "\033[1m"
	NC     = "\033[0m" // No Color
)

const moduleName = "logistics"

func init() {
	// Disable colors on Windows (unless using Windows Terminal)
	if runtime.GOOS == "windows" {
		if os.Getenv("WT_SESSION") == "" && os.Getenv("TERM_PROGRAM") != "vscode" {
			RED, GREEN, YELLOW, BLUE, CYAN, GRAY, BOLD, NC = "", "", "", "", "", "", "", ""
		}
	}
}

// Category represents a directory category
type Category struct {
	Name        string
	Prefix      string
	Depth       int
	Dirs        map[string]bool
	Description string
}

// ServiceStats holds statistics for a single service
type ServiceStats struct {
	Name         string
	Path         string
	TotalDeps    int
	InternalDeps int
	Categories   map[string]int
	OtherDeps    []string
	DepTree      map[string][]string
}

// DepAnalyzer analyzes dependencies
type DepAnalyzer struct {
	visited      map[string]bool
	allDeps      []string
	depGraph     map[string][]string
	moduleName   string
	categories   []*Category
	otherDeps    []string
	serviceStats []*ServiceStats
}

// NewDepAnalyzer creates a new analyzer
func NewDepAnalyzer(module string) *DepAnalyzer {
	return &DepAnalyzer{
		visited:    make(map[string]bool),
		allDeps:    make([]string, 0),
		depGraph:   make(map[string][]string),
		moduleName: module,
		categories: []*Category{
			{Name: "Generated proto files", Prefix: "gen/", Depth: 4, Dirs: make(map[string]bool), Description: "Proto-generated Go code"},
			{Name: "Shared packages", Prefix: "pkg/", Depth: 2, Dirs: make(map[string]bool), Description: "Common utilities and libraries"},
			{Name: "Services", Prefix: "services/", Depth: 2, Dirs: make(map[string]bool), Description: "Microservices"},
			{Name: "Migrations", Prefix: "migrations", Depth: 1, Dirs: make(map[string]bool), Description: "Database migrations"},
		},
		otherDeps:    make([]string, 0),
		serviceStats: make([]*ServiceStats, 0),
	}
}

// Reset resets the analyzer state for new analysis
func (a *DepAnalyzer) Reset() {
	a.visited = make(map[string]bool)
	a.allDeps = make([]string, 0)
	a.depGraph = make(map[string][]string)
	a.otherDeps = make([]string, 0)
	for _, cat := range a.categories {
		cat.Dirs = make(map[string]bool)
	}
}

// getImports returns internal imports for a package
func (a *DepAnalyzer) getImports(pkg string) ([]string, error) {
	cmd := exec.Command("go", "list", "-f", `{{range .Imports}}{{.}}{{"\n"}}{{end}}`, pkg)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var imports []string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		imp := scanner.Text()
		if strings.HasPrefix(imp, a.moduleName+"/") {
			imports = append(imports, imp)
		}
	}
	sort.Strings(imports)
	return imports, nil
}

// getInitialPackages returns packages for a path
func (a *DepAnalyzer) getInitialPackages(path string) ([]string, error) {
	cmd := exec.Command("go", "list", path)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var pkgs []string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		pkgs = append(pkgs, scanner.Text())
	}
	return pkgs, nil
}

// analyze performs BFS analysis on packages
func (a *DepAnalyzer) analyze(pkgs []string) error {
	queue := make([]string, len(pkgs))
	copy(queue, pkgs)

	iteration := 0
	for len(queue) > 0 {
		iteration++
		newDeps := 0

		// Process current queue
		currentQueue := make([]string, len(queue))
		copy(currentQueue, queue)
		queue = queue[:0]

		for _, pkg := range currentQueue {
			if a.visited[pkg] {
				continue
			}
			a.visited[pkg] = true
			a.allDeps = append(a.allDeps, pkg)

			imports, err := a.getImports(pkg)
			if err != nil {
				continue
			}

			a.depGraph[pkg] = imports

			for _, imp := range imports {
				if !a.visited[imp] {
					queue = append(queue, imp)
					newDeps++
				}
			}
		}

		if newDeps > 0 {
			fmt.Printf("  Iteration %d: found %s%d%s new dependencies\n", iteration, GREEN, newDeps, NC)
		}
	}

	sort.Strings(a.allDeps)
	a.categorize()
	return nil
}

// categorize groups dependencies by category
func (a *DepAnalyzer) categorize() {
	for _, dep := range a.allDeps {
		rel := strings.TrimPrefix(dep, a.moduleName+"/")
		categorized := false

		for _, cat := range a.categories {
			if strings.HasPrefix(rel, cat.Prefix) {
				parts := strings.Split(rel, "/")
				depth := cat.Depth
				if depth > len(parts) {
					depth = len(parts)
				}
				dir := strings.Join(parts[:depth], "/")
				cat.Dirs[dir] = true
				categorized = true
				break
			}
		}

		if !categorized {
			a.otherDeps = append(a.otherDeps, rel)
		}
	}
}

// analyzeService analyzes a single service and returns stats
func (a *DepAnalyzer) analyzeService(servicePath string) (*ServiceStats, error) {
	a.Reset()

	pkgs, err := a.getInitialPackages(servicePath)
	if err != nil {
		return nil, err
	}

	if err := a.analyze(pkgs); err != nil {
		return nil, err
	}

	// Collect stats
	stats := &ServiceStats{
		Name:         filepath.Base(strings.TrimSuffix(servicePath, "/...")),
		Path:         servicePath,
		TotalDeps:    len(a.allDeps),
		InternalDeps: 0,
		Categories:   make(map[string]int),
		OtherDeps:    make([]string, len(a.otherDeps)),
		DepTree:      make(map[string][]string),
	}

	copy(stats.OtherDeps, a.otherDeps)

	for _, cat := range a.categories {
		stats.Categories[cat.Name] = len(cat.Dirs)
		stats.InternalDeps += len(cat.Dirs)
	}

	// Copy dep tree
	for k, v := range a.depGraph {
		stats.DepTree[k] = v
	}

	return stats, nil
}

// findAllServices discovers all services in the services directory
func (a *DepAnalyzer) findAllServices() ([]string, error) {
	entries, err := os.ReadDir("services")
	if err != nil {
		return nil, err
	}

	var services []string
	for _, entry := range entries {
		if entry.IsDir() && strings.HasSuffix(entry.Name(), "-svc") {
			services = append(services, fmt.Sprintf("./services/%s/...", entry.Name()))
		}
	}
	sort.Strings(services)
	return services, nil
}

// printHeader prints the header
func printHeader() {
	fmt.Printf("%s╔═══════════════════════════════════════════════════════════════════╗%s\n", CYAN, NC)
	fmt.Printf("%s║       Recursive Dependency Analyzer v2.0                          ║%s\n", CYAN, NC)
	fmt.Printf("%s╚═══════════════════════════════════════════════════════════════════╝%s\n", CYAN, NC)
	fmt.Println()
}

// printFooter prints the footer
func printFooter() {
	fmt.Println()
	fmt.Printf("%s╔═══════════════════════════════════════════════════════════════════╗%s\n", CYAN, NC)
	fmt.Printf("%s║       Analysis Complete                                           ║%s\n", CYAN, NC)
	fmt.Printf("%s╚═══════════════════════════════════════════════════════════════════╝%s\n", CYAN, NC)
}

// printServiceStats prints statistics for a single service
func (a *DepAnalyzer) printServiceStats(stats *ServiceStats, detailed bool) {
	fmt.Printf("\n%s━━━ %s%s%s ━━━%s\n", BLUE, BOLD, stats.Name, NC+BLUE, NC)
	fmt.Printf("  Path: %s%s%s\n", GRAY, stats.Path, NC)
	fmt.Printf("  Total packages: %s%d%s\n", GREEN, stats.TotalDeps, NC)

	if detailed {
		fmt.Println("\n  Categories:")
		for _, cat := range a.categories {
			count := stats.Categories[cat.Name]
			if count > 0 {
				fmt.Printf("    %s✓%s %-25s %s%d%s directories\n", GREEN, NC, cat.Name, YELLOW, count, NC)
			}
		}

		if len(stats.OtherDeps) > 0 {
			fmt.Printf("    %s⚠%s %-25s %s%d%s uncategorized\n", YELLOW, NC, "Other", RED, len(stats.OtherDeps), NC)
			for _, dep := range stats.OtherDeps {
				fmt.Printf("      %s- %s%s\n", GRAY, dep, NC)
			}
		}
	}
}

// printTree prints dependency tree
func (a *DepAnalyzer) printTree(pkg string, prefix string, isLast bool, visited map[string]bool, depth, maxDepth int) {
	if depth > maxDepth {
		return
	}

	rel := strings.TrimPrefix(pkg, a.moduleName+"/")

	// Choose connector
	connector := "├── "
	if isLast {
		connector = "└── "
	}

	// Check for circular dependency
	if visited[pkg] {
		fmt.Printf("%s%s%s%s (circular)%s\n", prefix, connector, GRAY, rel, NC)
		return
	}
	visited[pkg] = true
	defer func() { visited[pkg] = false }()

	// Print current node
	fmt.Printf("%s%s%s\n", prefix, connector, rel)

	// Get dependencies
	deps := a.depGraph[pkg]
	if len(deps) == 0 {
		return
	}

	// New prefix for children
	newPrefix := prefix
	if isLast {
		newPrefix += "    "
	} else {
		newPrefix += "│   "
	}

	// Print children
	for i, dep := range deps {
		isChildLast := i == len(deps)-1
		a.printTree(dep, newPrefix, isChildLast, visited, depth+1, maxDepth)
	}
}

// printDependencyTree prints the full dependency tree for analyzed packages
func (a *DepAnalyzer) printDependencyTree(pkgs []string, maxDepth int) {
	fmt.Printf("\n%s=== Dependency Tree (max depth: %d) ===%s\n", GREEN, maxDepth, NC)

	for i, pkg := range pkgs {
		if i >= 5 { // Limit to first 5 packages
			fmt.Printf("\n%s... and %d more packages%s\n", GRAY, len(pkgs)-5, NC)
			break
		}
		fmt.Println()
		visited := make(map[string]bool)
		a.printTree(pkg, "", true, visited, 0, maxDepth)
	}
}

// printDockerCopy prints Dockerfile COPY commands
func (a *DepAnalyzer) printDockerCopy() {
	fmt.Printf("\n%s=== Dockerfile COPY Commands ===%s\n", GREEN, NC)

	for _, cat := range a.categories {
		if len(cat.Dirs) == 0 {
			continue
		}

		fmt.Printf("\n%s# %s%s\n", GRAY, cat.Name, NC)
		dirs := make([]string, 0, len(cat.Dirs))
		for dir := range cat.Dirs {
			dirs = append(dirs, dir)
		}
		sort.Strings(dirs)

		for _, dir := range dirs {
			fmt.Printf("COPY %s/ ./%s/\n", dir, dir)
		}
	}

	if len(a.otherDeps) > 0 {
		fmt.Printf("\n%s# Other (not categorized) - REVIEW REQUIRED!%s\n", YELLOW, NC)

		otherDirs := make(map[string]bool)
		for _, dep := range a.otherDeps {
			parts := strings.Split(dep, "/")
			var dir string
			if len(parts) >= 2 {
				dir = parts[0] + "/" + parts[1]
			} else {
				dir = parts[0]
			}
			otherDirs[dir] = true
		}

		dirs := make([]string, 0, len(otherDirs))
		for dir := range otherDirs {
			dirs = append(dirs, dir)
		}
		sort.Strings(dirs)

		for _, dir := range dirs {
			fmt.Printf("COPY %s/ ./%s/\n", dir, dir)
		}

		fmt.Printf("\n%s# Detailed 'Other' dependencies:%s\n", GRAY, NC)
		for _, dep := range a.otherDeps {
			fmt.Printf("%s#   - %s%s\n", GRAY, dep, NC)
		}
	}
}

// printAllDeps prints all dependencies
func (a *DepAnalyzer) printAllDeps() {
	fmt.Printf("\n%s=== All Internal Dependencies ===%s\n", GREEN, NC)

	for _, dep := range a.allDeps {
		rel := strings.TrimPrefix(dep, a.moduleName+"/")
		fmt.Printf("  %s\n", rel)
	}
}

// printTopDirs prints top-level directories
func (a *DepAnalyzer) printTopDirs() {
	fmt.Printf("\n%s=== Required Top-Level Directories ===%s\n", GREEN, NC)

	topDirs := make(map[string]int)
	for _, dep := range a.allDeps {
		rel := strings.TrimPrefix(dep, a.moduleName+"/")
		parts := strings.Split(rel, "/")
		if len(parts) > 0 {
			topDirs[parts[0]]++
		}
	}

	dirs := make([]string, 0, len(topDirs))
	for dir := range topDirs {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)

	for _, dir := range dirs {
		fmt.Printf("  %s%-15s%s (%s%d%s packages)\n", BOLD, dir+"/", NC, YELLOW, topDirs[dir], NC)
	}
}

// printSummary prints final summary
func (a *DepAnalyzer) printSummary() {
	fmt.Printf("\n%s=== Summary ===%s\n", GREEN, NC)
	fmt.Println()
	fmt.Println("Categories found:")

	for _, cat := range a.categories {
		if len(cat.Dirs) > 0 {
			fmt.Printf("  %s✓%s %-30s %s%d%s directories\n", GREEN, NC, cat.Name, YELLOW, len(cat.Dirs), NC)
		}
	}

	if len(a.otherDeps) > 0 {
		fmt.Printf("  %s⚠%s %-30s %s%d%s uncategorized (review required)\n",
			YELLOW, NC, "Other", RED, len(a.otherDeps), NC)
	}
}

// printGlobalSummary prints summary for all services
func (a *DepAnalyzer) printGlobalSummary() {
	if len(a.serviceStats) == 0 {
		return
	}

	fmt.Printf("\n%s╔═══════════════════════════════════════════════════════════════════╗%s\n", GREEN, NC)
	fmt.Printf("%s║       Global Statistics                                           ║%s\n", GREEN, NC)
	fmt.Printf("%s╚═══════════════════════════════════════════════════════════════════╝%s\n", GREEN, NC)

	// Table header
	fmt.Printf("\n%-25s %10s %10s %10s %10s %10s\n",
		"Service", "Total", "gen/", "pkg/", "services/", "Other")
	fmt.Println(strings.Repeat("─", 80))

	totalDeps := 0
	for _, stats := range a.serviceStats {
		fmt.Printf("%-25s %10d %10d %10d %10d %10d\n",
			stats.Name,
			stats.TotalDeps,
			stats.Categories["Generated proto files"],
			stats.Categories["Shared packages"],
			stats.Categories["Services"],
			len(stats.OtherDeps))
		totalDeps += stats.TotalDeps
	}

	fmt.Println(strings.Repeat("─", 80))
	fmt.Printf("%-25s %s%10d%s\n", "TOTAL (with duplicates)", BOLD, totalDeps, NC)
}

func main() {
	// Flags
	servicePath := flag.String("path", "", "Service path(s) to analyze (comma-separated, default: all services)")
	showTree := flag.Bool("tree", false, "Show dependency tree")
	treeDepth := flag.Int("depth", 3, "Dependency tree max depth")
	dockerOnly := flag.Bool("docker", false, "Show only Dockerfile COPY commands")
	allDeps := flag.Bool("all", false, "Show all dependencies list")
	detailed := flag.Bool("detailed", true, "Show detailed per-service statistics")
	flag.Parse()

	printHeader()

	analyzer := NewDepAnalyzer(moduleName)

	// Determine which services to analyze
	var servicePaths []string
	if *servicePath == "" {
		// Find all services
		fmt.Printf("%s[1/4] Discovering services...%s\n", BLUE, NC)
		services, err := analyzer.findAllServices()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%sError finding services: %v%s\n", RED, err, NC)
			os.Exit(1)
		}
		servicePaths = services
		fmt.Printf("  Found %s%d%s services\n", GREEN, len(services), NC)
		for _, svc := range services {
			fmt.Printf("    %s•%s %s\n", CYAN, NC, svc)
		}
	} else {
		// Parse comma-separated paths
		servicePaths = strings.Split(*servicePath, ",")
		for i := range servicePaths {
			servicePaths[i] = strings.TrimSpace(servicePaths[i])
		}
	}

	// Analyze each service
	fmt.Printf("\n%s[2/4] Analyzing dependencies...%s\n", BLUE, NC)

	for _, svcPath := range servicePaths {
		fmt.Printf("\n%sAnalyzing:%s %s\n", YELLOW, NC, svcPath)

		stats, err := analyzer.analyzeService(svcPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %sError: %v%s\n", RED, err, NC)
			continue
		}

		analyzer.serviceStats = append(analyzer.serviceStats, stats)
		analyzer.printServiceStats(stats, *detailed)
	}

	// Combined analysis (merge all dependencies)
	fmt.Printf("\n%s[3/4] Generating combined analysis...%s\n", BLUE, NC)

	analyzer.Reset()
	var allPkgs []string
	for _, svcPath := range servicePaths {
		pkgs, err := analyzer.getInitialPackages(svcPath)
		if err != nil {
			continue
		}
		allPkgs = append(allPkgs, pkgs...)
	}

	if err := analyzer.analyze(allPkgs); err != nil {
		fmt.Fprintf(os.Stderr, "%sError in combined analysis: %v%s\n", RED, err, NC)
	}

	// Output
	fmt.Printf("\n%s[4/4] Generating report...%s\n", BLUE, NC)

	if *allDeps {
		analyzer.printAllDeps()
	}

	analyzer.printTopDirs()

	if !*dockerOnly {
		analyzer.printDockerCopy()
		analyzer.printSummary()
	} else {
		analyzer.printDockerCopy()
	}

	if *showTree && len(allPkgs) > 0 {
		analyzer.printDependencyTree(allPkgs, *treeDepth)
	}

	// Global summary
	analyzer.printGlobalSummary()

	printFooter()
}
