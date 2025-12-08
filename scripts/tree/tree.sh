#!/bin/bash
# scripts/tree/tree.sh
# Directory tree printer (excludes .git contents)

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
MAX_DEPTH=-1
DIRS_ONLY=false
SHOW_HIDDEN=true
OUTPUT_FILE=""
NO_HEADER=false
NO_STATS=false
EXCLUDE_DIRS=".git"

# Stats
DIR_COUNT=0
FILE_COUNT=0

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --depth)
            MAX_DEPTH="$2"
            shift 2
            ;;
        --dirs)
            DIRS_ONLY=true
            shift
            ;;
        --no-hidden)
            SHOW_HIDDEN=false
            shift
            ;;
        --output|-o)
            OUTPUT_FILE="$2"
            shift 2
            ;;
        --no-header)
            NO_HEADER=true
            shift
            ;;
        --no-stats)
            NO_STATS=true
            shift
            ;;
        --exclude)
            EXCLUDE_DIRS="$2"
            shift 2
            ;;
        --help|-h)
            echo "Usage: $0 [OPTIONS] [PATH]"
            echo ""
            echo "Options:"
            echo "  --depth N       Maximum depth (-1 for unlimited)"
            echo "  --dirs          Show directories only"
            echo "  --no-hidden     Hide hidden files"
            echo "  --output, -o    Output file"
            echo "  --no-header     Don't print header"
            echo "  --no-stats      Don't print statistics"
            echo "  --exclude       Comma-separated dirs to exclude contents"
            echo "  --help, -h      Show this help"
            exit 0
            ;;
        *)
            ROOT_PATH="$1"
            shift
            ;;
    esac
done

# Disable colors for file output
if [ -n "$OUTPUT_FILE" ]; then
    BLUE="" CYAN="" GREEN="" YELLOW="" GRAY="" BOLD="" NC=""
    exec > "$OUTPUT_FILE"
fi

# Check if directory should be excluded
should_exclude_contents() {
    local name="$1"
    IFS=',' read -ra DIRS <<< "$EXCLUDE_DIRS"
    for dir in "${DIRS[@]}"; do
        if [ "$name" = "$(echo "$dir" | xargs)" ]; then
            return 0
        fi
    done
    return 1
}

# Print tree recursively
print_tree() {
    local path="$1"
    local prefix="$2"
    local depth="$3"

    # Check depth limit
    if [ "$MAX_DEPTH" -ge 0 ] && [ "$depth" -gt "$MAX_DEPTH" ]; then
        return
    fi

    # Get entries
    local entries=()
    while IFS= read -r -d '' entry; do
        entries+=("$entry")
    done < <(find "$path" -maxdepth 1 -mindepth 1 -print0 2>/dev/null | sort -z)

    # Filter entries
    local filtered=()
    for entry in "${entries[@]}"; do
        local name=$(basename "$entry")

        # Skip hidden if requested
        if [ "$SHOW_HIDDEN" = false ] && [[ "$name" == .* ]]; then
            continue
        fi

        # Skip files if dirs only
        if [ "$DIRS_ONLY" = true ] && [ ! -d "$entry" ]; then
            continue
        fi

        filtered+=("$entry")
    done

    # Sort: directories first
    local dirs=()
    local files=()
    for entry in "${filtered[@]}"; do
        if [ -d "$entry" ]; then
            dirs+=("$entry")
        else
            files+=("$entry")
        fi
    done

    local all_entries=("${dirs[@]}" "${files[@]}")
    local count=${#all_entries[@]}
    local i=0

    for entry in "${all_entries[@]}"; do
        i=$((i + 1))
        local name=$(basename "$entry")
        local is_last=$( [ $i -eq $count ] && echo true || echo false )
        local connector="├── "
        [ "$is_last" = true ] && connector="└── "

        if [ -d "$entry" ]; then
            echo -e "${prefix}${connector}${BLUE}${name}/${NC}"
            DIR_COUNT=$((DIR_COUNT + 1))

            # Check if contents should be excluded
            if should_exclude_contents "$name"; then
                continue
            fi

            local new_prefix="${prefix}│   "
            [ "$is_last" = true ] && new_prefix="${prefix}    "

            print_tree "$entry" "$new_prefix" $((depth + 1))
        else
            echo -e "${prefix}${connector}${name}"
            FILE_COUNT=$((FILE_COUNT + 1))
        fi
    done
}

# Header
if [ "$NO_HEADER" = false ]; then
    echo ""
    echo -e "${CYAN}╔═══════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║       Directory Tree                                              ║${NC}"
    echo -e "${CYAN}╚═══════════════════════════════════════════════════════════════════╝${NC}"
    echo ""
fi

# Print root
ROOT_NAME=$(basename "$(cd "$ROOT_PATH" && pwd)")
echo -e "${BOLD}${BLUE}${ROOT_NAME}/${NC}"

# Print tree
print_tree "$ROOT_PATH" "" 0

# Stats
if [ "$NO_STATS" = false ]; then
    echo ""
    echo -e "${GRAY}${DIR_COUNT} directories, ${FILE_COUNT} files${NC}"
fi
