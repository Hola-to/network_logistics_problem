#Requires -Version 5.1
<#
.SYNOPSIS
    Directory tree printer

.PARAMETER Path
    Root directory path (default: current directory)

.PARAMETER Depth
    Maximum depth (-1 for unlimited)

.PARAMETER DirsOnly
    Show directories only

.PARAMETER NoHidden
    Hide hidden files/directories

.PARAMETER Output
    Output file path

.PARAMETER Exclude
    Comma-separated directories to exclude contents (default: .git)
#>

param(
    [string]$Path = ".",
    [int]$Depth = -1,
    [switch]$DirsOnly,
    [switch]$NoHidden,
    [string]$Output = "",
    [switch]$NoHeader,
    [switch]$NoStats,
    [string]$Exclude = ".git"
)

# Stats
$script:DirCount = 0
$script:FileCount = 0

# Exclude list
$ExcludeList = $Exclude -split ',' | ForEach-Object { $_.Trim() }

function Should-ExcludeContents {
    param([string]$Name)
    return $ExcludeList -contains $Name
}

function Print-Tree {
    param(
        [string]$CurrentPath,
        [string]$Prefix,
        [int]$CurrentDepth
    )

    if ($Depth -ge 0 -and $CurrentDepth -gt $Depth) { return }

    $entries = Get-ChildItem -Path $CurrentPath -Force -ErrorAction SilentlyContinue

    # Filter
    $filtered = $entries | Where-Object {
        $show = $true
        if ($NoHidden -and $_.Name.StartsWith('.')) { $show = $false }
        if ($DirsOnly -and -not $_.PSIsContainer) { $show = $false }
        $show
    } | Sort-Object { -not $_.PSIsContainer }, Name

    $count = @($filtered).Count
    $i = 0

    foreach ($entry in $filtered) {
        $i++
        $isLast = $i -eq $count
        $connector = if ($isLast) { "└── " } else { "├── " }

        if ($entry.PSIsContainer) {
            $line = "$Prefix$connector$($entry.Name)/"
            if ($Output) { Add-Content -Path $Output -Value $line }
            else { Write-Host $line -ForegroundColor Blue }
            $script:DirCount++

            if (-not (Should-ExcludeContents $entry.Name)) {
                $newPrefix = if ($isLast) { "$Prefix    " } else { "$Prefix│   " }
                Print-Tree -CurrentPath $entry.FullName -Prefix $newPrefix -CurrentDepth ($CurrentDepth + 1)
            }
        }
        else {
            $line = "$Prefix$connector$($entry.Name)"
            if ($Output) { Add-Content -Path $Output -Value $line }
            else { Write-Host $line }
            $script:FileCount++
        }
    }
}

# Clear output file
if ($Output) {
    Set-Content -Path $Output -Value ""
}

# Header
if (-not $NoHeader) {
    $header = @"

╔═══════════════════════════════════════════════════════════════════╗
║       Directory Tree                                              ║
╚═══════════════════════════════════════════════════════════════════╝

"@
    if ($Output) { Add-Content -Path $Output -Value $header }
    else { Write-Host $header -ForegroundColor Cyan }
}

# Root
$rootName = (Get-Item -Path $Path).Name
$rootLine = "$rootName/"
if ($Output) { Add-Content -Path $Output -Value $rootLine }
else { Write-Host $rootLine -ForegroundColor Blue }

# Tree
Print-Tree -CurrentPath $Path -Prefix "" -CurrentDepth 0

# Stats
if (-not $NoStats) {
    $stats = "`n$script:DirCount directories, $script:FileCount files"
    if ($Output) { Add-Content -Path $Output -Value $stats }
    else { Write-Host $stats -ForegroundColor DarkGray }
}
