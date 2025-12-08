#!/bin/bash
# Count lines of code

set -e

# Colors
BLUE='\033[0;34m'
CYAN='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
GRAY='\033[0;90m'
BOLD='\033[1m'
NC='\033[0m'

# Defaults
ROOT_PATH="."
DETAILED=true
SIMPLE=false
EXCLUDE=".git,vendor,node_modules,.idea,.vscode,.zed"

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --detailed)
            DETAILED=true
            shift
            ;;
        --simple|-s)
            SIMPLE=true
            shift
            ;;
        --exclude)
            EXCLUDE="$2"
            shift 2
            ;;
        --help|-h)
            echo "Usage: $0 [OPTIONS] [PATH]"
            echo ""
            echo "Options:"
            echo "  --detailed      Show detailed statistics (default)"
            echo "  --simple, -s    Print only total lines of code"
            echo "  --exclude       Comma-separated directories to exclude"
            echo "  --help, -h      Show this help"
            exit 0
            ;;
        *)
            ROOT_PATH="$1"
            shift
            ;;
    esac
done

# Build exclude pattern for find
EXCLUDE_PATTERN=""
IFS=',' read -ra EXCLUDE_ARRAY <<< "$EXCLUDE"
for dir in "${EXCLUDE_ARRAY[@]}"; do
    dir=$(echo "$dir" | xargs)
    if [ -n "$EXCLUDE_PATTERN" ]; then
        EXCLUDE_PATTERN="$EXCLUDE_PATTERN -o"
    fi
    EXCLUDE_PATTERN="$EXCLUDE_PATTERN -path \"*/$dir/*\" -o -path \"*/$dir\""
done

# Count lines for files
count_lines() {
    local pattern="$1"
    find "$ROOT_PATH" -type f -name "$pattern" \
        ! \( $EXCLUDE_PATTERN \) \
        -exec cat {} \; 2>/dev/null | wc -l | tr -d ' '
}

# Count files
count_files() {
    local pattern="$1"
    find "$ROOT_PATH" -type f -name "$pattern" \
        ! \( $EXCLUDE_PATTERN \) 2>/dev/null | wc -l | tr -d ' '
}

# Extensions to count
declare -A EXT_FILES EXT_LINES
EXTENSIONS=("*.go" "*.proto" "*.sql" "*.yaml" "*.yml" "*.json" "*.toml" "*.md" "*.sh" "*.ps1")

# Count all
TOTAL_FILES=0
TOTAL_LINES=0

for ext in "${EXTENSIONS[@]}"; do
    files=$(count_files "$ext")
    lines=$(count_lines "$ext")

    if [ "$files" -gt 0 ]; then
        ext_name="${ext#\*}"
        EXT_FILES["$ext_name"]=$files
        EXT_LINES["$ext_name"]=$lines
        TOTAL_FILES=$((TOTAL_FILES + files))
        TOTAL_LINES=$((TOTAL_LINES + lines))
    fi
done

# Simple output
if [ "$SIMPLE" = true ]; then
    echo "$TOTAL_LINES"
    exit 0
fi

# Detailed output
echo -e "\n${CYAN}╔═══════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║       Lines of Code Counter                                       ║${NC}"
echo -e "${CYAN}╚═══════════════════════════════════════════════════════════════════╝${NC}"
echo ""

if [ "$DETAILED" = true ]; then
    echo -e "${GREEN}=== By Extension ===${NC}\n"
    printf "%-12s %10s %12s\n" "Extension" "Files" "Lines"
    printf "%s\n" "$(printf '─%.0s' {1..36})"

    # Sort by lines (descending)
    for ext in $(for k in "${!EXT_LINES[@]}"; do echo "$k ${EXT_LINES[$k]}"; done | sort -t' ' -k2 -rn | cut -d' ' -f1); do
        printf "%-12s %10s %12s\n" "$ext" "${EXT_FILES[$ext]}" "${EXT_LINES[$ext]}"
    done

    printf "%s\n" "$(printf '─%.0s' {1..36})"
    printf "${BOLD}%-12s %10s %12s${NC}\n" "TOTAL" "$TOTAL_FILES" "$TOTAL_LINES"
fi

# Summary
echo -e "\n${GREEN}=== Summary ===${NC}\n"
echo -e "  Total Files: ${YELLOW}${TOTAL_FILES}${NC}"
echo -e "  Total Lines: ${BOLD}${GREEN}${TOTAL_LINES}${NC}"

echo ""
