#Requires -Version 5.1
<#
.SYNOPSIS
    Recursive Dependency Analyzer for Go projects (Windows PowerShell)

.DESCRIPTION
    Analyzes internal dependencies for Go services and generates Dockerfile COPY commands.

.PARAMETER ServicePaths
    Service paths to analyze. If not specified, all services are analyzed.

.PARAMETER ShowTree
    Show dependency tree.

.PARAMETER TreeDepth
    Maximum depth for dependency tree (default: 3).

.PARAMETER DockerOnly
    Show only Dockerfile COPY commands.

.EXAMPLE
    .\list-deps.ps1
    Analyze all services.

.EXAMPLE
    .\list-deps.ps1 -ServicePaths "./services/simulation-svc/..."
    Analyze single service.

.EXAMPLE
    .\list-deps.ps1 -ShowTree -TreeDepth 2
    Show dependency tree with depth 2.
#>

param(
    [string[]]$ServicePaths = @(),
    [switch]$ShowTree,
    [int]$TreeDepth = 3,
    [switch]$DockerOnly,
    [switch]$Help
)

# Module name
$ModuleName = "logistics"

# Colors
$Colors = @{
    Red    = "Red"
    Green  = "Green"
    Yellow = "Yellow"
    Blue   = "Blue"
    Cyan   = "Cyan"
    Gray   = "DarkGray"
    White  = "White"
}

function Write-ColorOutput {
    param(
        [string]$Message,
        [string]$Color = "White"
    )
    Write-Host $Message -ForegroundColor $Color
}

function Write-Header {
    Write-ColorOutput "╔═══════════════════════════════════════════════════════════════════╗" -Color Cyan
    Write-ColorOutput "║       Recursive Dependency Analyzer v2.0 (PowerShell)             ║" -Color Cyan
    Write-ColorOutput "╚═══════════════════════════════════════════════════════════════════╝" -Color Cyan
    Write-Host ""
}

function Write-Footer {
    Write-Host ""
    Write-ColorOutput "╔═══════════════════════════════════════════════════════════════════╗" -Color Cyan
    Write-ColorOutput "║       Analysis Complete                                           ║" -Color Cyan
    Write-ColorOutput "╚═══════════════════════════════════════════════════════════════════╝" -Color Cyan
}

function Get-Imports {
    param([string]$Package)

    try {
        $output = go list -f '{{range .Imports}}{{.}}{{"`n"}}{{end}}' $Package 2>$null
        if ($LASTEXITCODE -ne 0) { return @() }

        $imports = $output -split "`n" | Where-Object { $_ -match "^$ModuleName/" } | Sort-Object -Unique
        return $imports
    }
    catch {
        return @()
    }
}

function Get-InitialPackages {
    param([string]$Path)

    try {
        $output = go list $Path 2>$null
        if ($LASTEXITCODE -ne 0) { return @() }

        return $output -split "`n" | Where-Object { $_ -ne "" }
    }
    catch {
        return @()
    }
}

function Find-AllServices {
    $services = Get-ChildItem -Path "services" -Directory |
        Where-Object { $_.Name -match "-svc$" } |
        ForEach-Object { "./services/$($_.Name)/..." }

    return $services | Sort-Object
}

function Analyze-Service {
    param(
        [string]$ServicePath,
        [hashtable]$GlobalStats
    )

    $serviceName = Split-Path -Leaf ($ServicePath -replace '/\.\.\.', '')

    Write-Host ""
    Write-ColorOutput "━━━ $serviceName ━━━" -Color Cyan
    Write-ColorOutput "  Path: $ServicePath" -Color Gray

    # Get initial packages
    $initialPkgs = Get-InitialPackages -Path $ServicePath
    if ($initialPkgs.Count -eq 0) {
        Write-ColorOutput "  Error: No packages found" -Color Red
        return $null
    }

    # BFS
    $visited = @{}
    $allDeps = [System.Collections.Generic.List[string]]::new()
    $depGraph = @{}
    $queue = [System.Collections.Generic.Queue[string]]::new()

    foreach ($pkg in $initialPkgs) {
        $queue.Enqueue($pkg)
    }

    $iteration = 0
    while ($queue.Count -gt 0) {
        $iteration++
        $currentQueue = @()
        while ($queue.Count -gt 0) {
            $currentQueue += $queue.Dequeue()
        }

        $newDeps = 0
        foreach ($pkg in $currentQueue) {
            if ($visited.ContainsKey($pkg)) { continue }

            $visited[$pkg] = $true
            $allDeps.Add($pkg)

            $imports = Get-Imports -Package $pkg
            $depGraph[$pkg] = $imports

            foreach ($imp in $imports) {
                if (-not $visited.ContainsKey($imp)) {
                    $queue.Enqueue($imp)
                    $newDeps++
                }
            }
        }

        if ($newDeps -gt 0) {
            Write-Host "  Iteration ${iteration}: found " -NoNewline
            Write-Host $newDeps -ForegroundColor Green -NoNewline
            Write-Host " new dependencies"
        }
    }

    $allDeps = $allDeps | Sort-Object -Unique

    # Count categories
    $genCount = ($allDeps | Where-Object { $_ -match "^$ModuleName/gen/" }).Count
    $pkgCount = ($allDeps | Where-Object { $_ -match "^$ModuleName/pkg/" }).Count
    $svcCount = ($allDeps | Where-Object { $_ -match "^$ModuleName/services/" }).Count
    $migCount = ($allDeps | Where-Object { $_ -match "^$ModuleName/migrations" }).Count
    $otherCount = $allDeps.Count - $genCount - $pkgCount - $svcCount - $migCount

    Write-Host "  Total packages: " -NoNewline
    Write-Host $allDeps.Count -ForegroundColor Green
    Write-Host ""
    Write-Host "  Categories:"

    if ($genCount -gt 0) {
        Write-Host "    " -NoNewline
        Write-Host "✓" -ForegroundColor Green -NoNewline
        Write-Host " Generated proto files    " -NoNewline
        Write-Host $genCount -ForegroundColor Yellow
    }
    if ($pkgCount -gt 0) {
        Write-Host "    " -NoNewline
        Write-Host "✓" -ForegroundColor Green -NoNewline
        Write-Host " Shared packages          " -NoNewline
        Write-Host $pkgCount -ForegroundColor Yellow
    }
    if ($svcCount -gt 0) {
        Write-Host "    " -NoNewline
        Write-Host "✓" -ForegroundColor Green -NoNewline
        Write-Host " Services                 " -NoNewline
        Write-Host $svcCount -ForegroundColor Yellow
    }
    if ($migCount -gt 0) {
        Write-Host "    " -NoNewline
        Write-Host "✓" -ForegroundColor Green -NoNewline
        Write-Host " Migrations               " -NoNewline
        Write-Host $migCount -ForegroundColor Yellow
    }
    if ($otherCount -gt 0) {
        Write-Host "    " -NoNewline
        Write-Host "⚠" -ForegroundColor Yellow -NoNewline
        Write-Host " Other                    " -NoNewline
        Write-Host $otherCount -ForegroundColor Red
    }

    return @{
        Name = $serviceName
        Path = $ServicePath
        Total = $allDeps.Count
        Gen = $genCount
        Pkg = $pkgCount
        Svc = $svcCount
        Mig = $migCount
        Other = $otherCount
        Deps = $allDeps
        DepGraph = $depGraph
    }
}

function Print-DockerCopy {
    param([string[]]$AllDeps)

    Write-Host ""
    Write-ColorOutput "=== Dockerfile COPY Commands ===" -Color Green

    # Gen
    $genDirs = $AllDeps |
        Where-Object { $_ -match "^$ModuleName/gen/" } |
        ForEach-Object { ($_ -replace "^$ModuleName/", "") -split '/' | Select-Object -First 4 | Join-String -Separator '/' } |
        Sort-Object -Unique

    if ($genDirs) {
        Write-Host ""
        Write-ColorOutput "# Generated proto files" -Color Gray
        foreach ($dir in $genDirs) {
            Write-Host "COPY $dir/ ./$dir/"
        }
    }

    # Pkg
    $pkgDirs = $AllDeps |
        Where-Object { $_ -match "^$ModuleName/pkg/" } |
        ForEach-Object { ($_ -replace "^$ModuleName/", "") -split '/' | Select-Object -First 2 | Join-String -Separator '/' } |
        Sort-Object -Unique

    if ($pkgDirs) {
        Write-Host ""
        Write-ColorOutput "# Shared packages" -Color Gray
        foreach ($dir in $pkgDirs) {
            Write-Host "COPY $dir/ ./$dir/"
        }
    }

    # Services
    $svcDirs = $AllDeps |
        Where-Object { $_ -match "^$ModuleName/services/" } |
        ForEach-Object { ($_ -replace "^$ModuleName/", "") -split '/' | Select-Object -First 2 | Join-String -Separator '/' } |
        Sort-Object -Unique

    if ($svcDirs) {
        Write-Host ""
        Write-ColorOutput "# Services" -Color Gray
        foreach ($dir in $svcDirs) {
            Write-Host "COPY $dir/ ./$dir/"
        }
    }

    # Migrations
    $migDirs = $AllDeps |
        Where-Object { $_ -match "^$ModuleName/migrations" } |
        ForEach-Object { ($_ -replace "^$ModuleName/", "") -split '/' | Select-Object -First 1 } |
        Sort-Object -Unique

    if ($migDirs) {
        Write-Host ""
        Write-ColorOutput "# Migrations" -Color Gray
        foreach ($dir in $migDirs) {
            Write-Host "COPY $dir/ ./$dir/"
        }
    }
}

function Print-GlobalStats {
    param([array]$Stats)

    Write-Host ""
    Write-ColorOutput "╔═══════════════════════════════════════════════════════════════════╗" -Color Green
    Write-ColorOutput "║       Global Statistics                                           ║" -Color Green
    Write-ColorOutput "╚═══════════════════════════════════════════════════════════════════╝" -Color Green
    Write-Host ""

    $format = "{0,-25} {1,10} {2,10} {3,10} {4,10} {5,10}"
    Write-Host ($format -f "Service", "Total", "gen/", "pkg/", "services/", "Other")
    Write-Host ("-" * 80)

    $totalAll = 0
    foreach ($stat in $Stats) {
        Write-Host ($format -f $stat.Name, $stat.Total, $stat.Gen, $stat.Pkg, $stat.Svc, $stat.Other)
        $totalAll += $stat.Total
    }

    Write-Host ("-" * 80)
    $line = $format -f "TOTAL (with duplicates)", $totalAll, "", "", "", ""
    Write-Host $line -ForegroundColor White
}

function Print-Tree {
    param(
        [string]$Package,
        [string]$Prefix,
        [int]$Depth,
        [int]$MaxDepth,
        [hashtable]$DepGraph,
        [hashtable]$Visited
    )

    if ($Depth -gt $MaxDepth) { return }

    $rel = $Package -replace "^$ModuleName/", ""

    if ($Visited.ContainsKey($Package)) {
        Write-Host "$Prefix└── " -NoNewline
        Write-ColorOutput "$rel (circular)" -Color Gray
        return
    }

    $Visited[$Package] = $true

    Write-Host "$Prefix├── $rel"

    $deps = $DepGraph[$Package]
    if (-not $deps -or $deps.Count -eq 0) { return }

    $deps = $deps | Select-Object -First 5
    $count = 0
    $total = $deps.Count

    foreach ($dep in $deps) {
        $count++
        $newPrefix = if ($count -eq $total) { "$Prefix    " } else { "$Prefix│   " }
        Print-Tree -Package $dep -Prefix $newPrefix -Depth ($Depth + 1) -MaxDepth $MaxDepth -DepGraph $DepGraph -Visited $Visited
    }
}

# === MAIN ===

if ($Help) {
    Get-Help $MyInvocation.MyCommand.Path -Detailed
    exit 0
}

Write-Header

# Find services
if ($ServicePaths.Count -eq 0) {
    Write-ColorOutput "[1/4] Discovering services..." -Color Blue
    $ServicePaths = Find-AllServices
    Write-Host "  Found " -NoNewline
    Write-Host $ServicePaths.Count -ForegroundColor Green -NoNewline
    Write-Host " services"

    foreach ($svc in $ServicePaths) {
        Write-Host "    " -NoNewline
        Write-Host "•" -ForegroundColor Cyan -NoNewline
        Write-Host " $svc"
    }
}

Write-Host ""
Write-ColorOutput "[2/4] Analyzing dependencies..." -Color Blue

# Analyze each service
$allStats = @()
$combinedDeps = [System.Collections.Generic.List[string]]::new()
$combinedGraph = @{}

foreach ($svcPath in $ServicePaths) {
    $result = Analyze-Service -ServicePath $svcPath
    if ($result) {
        $allStats += $result
        foreach ($dep in $result.Deps) {
            if (-not $combinedDeps.Contains($dep)) {
                $combinedDeps.Add($dep)
            }
        }
        foreach ($key in $result.DepGraph.Keys) {
            if (-not $combinedGraph.ContainsKey($key)) {
                $combinedGraph[$key] = $result.DepGraph[$key]
            }
        }
    }
}

$combinedDeps = $combinedDeps | Sort-Object -Unique

Write-Host ""
Write-ColorOutput "[3/4] Generating combined analysis..." -Color Blue

# Top-level directories
Write-Host ""
Write-ColorOutput "=== Required Top-Level Directories ===" -Color Green

$topDirs = $combinedDeps |
    ForEach-Object { ($_ -replace "^$ModuleName/", "") -split '/' | Select-Object -First 1 } |
    Group-Object |
    Sort-Object Count -Descending

foreach ($dir in $topDirs) {
    $name = $dir.Name + "/"
    Write-Host "  " -NoNewline
    Write-Host ("{0,-15}" -f $name) -NoNewline
    Write-Host " (" -NoNewline
    Write-Host $dir.Count -ForegroundColor Yellow -NoNewline
    Write-Host " packages)"
}

Write-Host ""
Write-ColorOutput "[4/4] Generating report..." -Color Blue

# Docker COPY
Print-DockerCopy -AllDeps $combinedDeps

# Summary
if (-not $DockerOnly) {
    Write-Host ""
    Write-ColorOutput "=== Summary ===" -Color Green
    Write-Host ""
    Write-Host "Categories found:"

    if (($combinedDeps | Where-Object { $_ -match "^$ModuleName/gen/" }).Count -gt 0) {
        Write-Host "  " -NoNewline
        Write-Host "✓" -ForegroundColor Green -NoNewline
        Write-Host " Generated proto files"
    }
    if (($combinedDeps | Where-Object { $_ -match "^$ModuleName/pkg/" }).Count -gt 0) {
        Write-Host "  " -NoNewline
        Write-Host "✓" -ForegroundColor Green -NoNewline
        Write-Host " Shared packages"
    }
    if (($combinedDeps | Where-Object { $_ -match "^$ModuleName/services/" }).Count -gt 0) {
        Write-Host "  " -NoNewline
        Write-Host "✓" -ForegroundColor Green -NoNewline
        Write-Host " Services"
    }
    if (($combinedDeps | Where-Object { $_ -match "^$ModuleName/migrations" }).Count -gt 0) {
        Write-Host "  " -NoNewline
        Write-Host "✓" -ForegroundColor Green -NoNewline
        Write-Host " Migrations"
    }
}

# Tree
if ($ShowTree) {
    Write-Host ""
    Write-ColorOutput "=== Dependency Tree (max depth: $TreeDepth) ===" -Color Green

    $treePkgs = $combinedDeps | Select-Object -First 5
    foreach ($pkg in $treePkgs) {
        Write-Host ""
        $visited = @{}
        Print-Tree -Package $pkg -Prefix "" -Depth 0 -MaxDepth $TreeDepth -DepGraph $combinedGraph -Visited $visited
    }
}

# Global stats
Print-GlobalStats -Stats $allStats

Write-Footer
