#Requires -Version 5.1
<#
.SYNOPSIS
    Count lines of code

.PARAMETER Path
    Root directory path

.PARAMETER Detailed
    Show detailed statistics

.PARAMETER Simple
    Print only total lines of code

.PARAMETER Exclude
    Comma-separated directories to exclude
#>

param(
    [string]$Path = ".",
    [switch]$Detailed = $true,
    [switch]$Simple,
    [string]$Exclude = ".git,vendor,node_modules,.idea,.vscode,.zed"
)

$ExcludeList = $Exclude -split ',' | ForEach-Object { $_.Trim() }

function Should-Exclude {
    param([string]$FilePath)
    foreach ($dir in $ExcludeList) {
        if ($FilePath -match "[\\/]$([regex]::Escape($dir))[\\/]?" -or $FilePath -match "^$([regex]::Escape($dir))[\\/]") {
            return $true
        }
    }
    return $false
}

# Extensions
$Extensions = @(".go", ".proto", ".sql", ".yaml", ".yml", ".json", ".toml", ".md", ".sh", ".ps1")
$Stats = @{}
$TotalFiles = 0
$TotalLines = 0

# Collect files
$allFiles = Get-ChildItem -Path $Path -Recurse -File -ErrorAction SilentlyContinue | Where-Object {
    $Extensions -contains $_.Extension -and -not (Should-Exclude $_.FullName)
}

foreach ($file in $allFiles) {
    $lines = (Get-Content $file.FullName -ErrorAction SilentlyContinue | Measure-Object -Line).Lines
    $ext = $file.Extension

    if (-not $Stats.ContainsKey($ext)) {
        $Stats[$ext] = @{ Files = 0; Lines = 0 }
    }

    $Stats[$ext].Files++
    $Stats[$ext].Lines += $lines
    $TotalFiles++
    $TotalLines += $lines
}

# Simple output
if ($Simple) {
    Write-Host $TotalLines
    exit 0
}

# Detailed output
Write-Host ""
Write-Host "╔═══════════════════════════════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║       Lines of Code Counter                                       ║" -ForegroundColor Cyan
Write-Host "╚═══════════════════════════════════════════════════════════════════╝" -ForegroundColor Cyan
Write-Host ""

if ($Detailed) {
    Write-Host "=== By Extension ===" -ForegroundColor Green
    Write-Host ""

    $format = "{0,-12} {1,10} {2,12}"
    Write-Host ($format -f "Extension", "Files", "Lines")
    Write-Host ("-" * 36)

    $Stats.GetEnumerator() | Sort-Object { $_.Value.Lines } -Descending | ForEach-Object {
        Write-Host ($format -f $_.Key, $_.Value.Files, $_.Value.Lines)
    }

    Write-Host ("-" * 36)
    Write-Host ($format -f "TOTAL", $TotalFiles, $TotalLines) -ForegroundColor White
}

Write-Host ""
Write-Host "=== Summary ===" -ForegroundColor Green
Write-Host ""
Write-Host "  Total Files: " -NoNewline
Write-Host $TotalFiles -ForegroundColor Yellow
Write-Host "  Total Lines: " -NoNewline
Write-Host $TotalLines -ForegroundColor Green

Write-Host ""
