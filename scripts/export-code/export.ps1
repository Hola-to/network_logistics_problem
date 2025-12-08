#Requires -Version 5.1
<#
.SYNOPSIS
    Export source code to markdown

.PARAMETER Dirs
    Comma-separated directories to export

.PARAMETER Output
    Output file path

.PARAMETER IncludeTests
    Include test files

.PARAMETER IncludeGenerated
    Include generated files
#>

param(
    [string]$Dirs = "api,pkg,services,migrations",
    [string]$Output = "logistics-code.md",
    [switch]$IncludeTests,
    [switch]$IncludeGenerated
)

# Language mapping
$LangMap = @{
    ".go"    = "go"
    ".proto" = "protobuf"
    ".sql"   = "sql"
    ".yaml"  = "yaml"
    ".yml"   = "yaml"
    ".json"  = "json"
    ".toml"  = "toml"
    ".md"    = "markdown"
    ".sh"    = "bash"
    ".ps1"   = "powershell"
    ".mod"   = "go"
    ".sum"   = "text"
}

# Exclude patterns
$ExcludePatterns = @("_test.go", ".pb.go", "_grpc.pb.go", ".connect.go", "mock_", "mocks", "testdata", "vendor")

if ($IncludeTests) {
    $ExcludePatterns = $ExcludePatterns | Where-Object { $_ -notmatch "test" }
}

if ($IncludeGenerated) {
    $ExcludePatterns = $ExcludePatterns | Where-Object { $_ -notmatch "\.pb\.go|\.connect\.go" }
}

function Should-Exclude {
    param([string]$Path)
    foreach ($pattern in $ExcludePatterns) {
        if ($Path -match [regex]::Escape($pattern)) {
            return $true
        }
    }
    return $false
}

# Header
Write-Host ""
Write-Host "╔═══════════════════════════════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║       Code Exporter                                               ║" -ForegroundColor Cyan
Write-Host "╚═══════════════════════════════════════════════════════════════════╝" -ForegroundColor Cyan
Write-Host ""

Write-Host "[1/3] Initializing..." -ForegroundColor Blue
Write-Host "  Directories: $Dirs" -ForegroundColor Yellow
Write-Host "  Output: $Output" -ForegroundColor Yellow

# Collect files
Write-Host ""
Write-Host "[2/3] Collecting files..." -ForegroundColor Blue

$dirArray = $Dirs -split ',' | ForEach-Object { $_.Trim() }
$allFiles = @()
$totalLines = 0

foreach ($dir in $dirArray) {
    if (-not (Test-Path $dir)) {
        Write-Host "  ⚠ Directory not found: $dir" -ForegroundColor Yellow
        continue
    }

    $files = Get-ChildItem -Path $dir -Recurse -File -Include "*.go", "*.proto", "*.sql", "*.yaml", "*.yml", "*.json", "*.toml", "*.sh", "*.md" -ErrorAction SilentlyContinue

    foreach ($file in $files) {
        if (Should-Exclude $file.FullName) { continue }
        if (-not $LangMap.ContainsKey($file.Extension)) { continue }

        $lines = (Get-Content $file.FullName -ErrorAction SilentlyContinue | Measure-Object -Line).Lines
        $allFiles += [PSCustomObject]@{
            Path = $file.FullName
            RelPath = $file.FullName.Replace((Get-Location).Path + "\", "").Replace("\", "/")
            Name = $file.Name
            Extension = $file.Extension
            Lines = $lines
        }
        $totalLines += $lines
    }
}

$allFiles = $allFiles | Sort-Object RelPath
Write-Host "  Found $($allFiles.Count) files ($totalLines lines)" -ForegroundColor Green

# Generate output
Write-Host ""
Write-Host "[3/3] Exporting to $Output..." -ForegroundColor Blue

$content = @"
# Logistics Platform - Source Code

> Generated: $(Get-Date -Format "yyyy-MM-dd HH:mm:ss")

## Table of Contents

"@

# TOC
$currentDir = ""
foreach ($file in $allFiles) {
    $dir = Split-Path $file.RelPath -Parent
    if ($dir -ne $currentDir) {
        $currentDir = $dir
        $anchor = $dir.ToLower().Replace("/", "-").Replace(".", "")
        $content += "- [$dir](#$anchor)`n"
    }
}

$content += @"

---

## Statistics

| Metric | Value |
|--------|-------|
| Total Files | $($allFiles.Count) |
| Total Lines | $totalLines |

---

## Source Files

"@

# Files
$currentDir = ""
foreach ($file in $allFiles) {
    $dir = Split-Path $file.RelPath -Parent

    if ($dir -ne $currentDir) {
        $currentDir = $dir
        $content += "`n### $dir`n`n"
    }

    $lang = $LangMap[$file.Extension]
    $fileContent = Get-Content $file.Path -Raw -ErrorAction SilentlyContinue

    $content += @"
#### ``$($file.Name)``

> Path: ``$($file.RelPath)`` | Lines: $($file.Lines)

``````$lang
$fileContent

"@
}
# Write file

Set-Content -Path $Output -Value $content -Encoding UTF8
# Summary

$fileSize = (Get-Item $Output).Length / 1KB

Write-Host ""
Write-Host "=== Export Complete ===" -ForegroundColor Green
Write-Host " Output: $Output" -ForegroundColor Cyan
Write-Host " Files: $($allFiles.Count)" -ForegroundColor Green
Write-Host " Lines: $totalLines" -ForegroundColor Green
Write-Host " Size: $([math]::Round($fileSize, 2)) KB" -ForegroundColor Green

Write-Host ""
Write-Host "╔═══════════════════════════════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║ Done! ║" -ForegroundColor Cyan
Write-Host "╚═══════════════════════════════════════════════════════════════════╝" -ForegroundColor Cyan
