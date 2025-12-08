#!/bin/bash
# scripts/export/export.sh
# Export source code to markdown

set -e

# Colors
BLUE='\033[0;34m'
CYAN='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
GRAY='\033[0;90m'
RED='\033[0;31m'
NC='\033[0m'

# Defaults
DIRS="api,pkg,services,migrations"
OUTPUT="logistics-code.md"
EXCLUDE_PATTERNS="_test.go|\.pb\.go|_grpc\.pb\.go|\.connect\.go|mock_|mocks/|testdata/|vendor/"
INCLUDE_TESTS=false
INCLUDE_GENERATED=false

# Extensions mapping
declare -A LANG_MAP=(
    ["go"]="go"
    ["proto"]="protobuf"
    ["sql"]="sql"
    ["yaml"]="yaml"
    ["yml"]="yaml"
    ["json"]="json"
    ["toml"]="toml"
    ["md"]="markdown"
    ["sh"]="bash"
    ["ps1"]="powershell"
    ["mod"]="go"
    ["sum"]="text"
)

# Stats
TOTAL_FILES=0
TOTAL_LINES=0

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --dirs|-d)
            DIRS="$2"
            shift 2
            ;;
        --output|-o)
            OUTPUT="$2"
            shift 2
            ;;
        --include-tests)
            INCLUDE_TESTS=true
            shift
            ;;
        --include-generated)
            INCLUDE_GENERATED=true
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --dirs, -d      Comma-separated directories (default: api,pkg,services,migrations)"
            echo "  --output, -o    Output file (default: logistics-code.md)"
            echo "  --include-tests Include test files"
            echo "  --include-generated Include generated files"
            echo "  --help, -h      Show this help"
            exit 0
            ;;
        *)
            shift
            ;;
    esac
done

# Update exclude patterns based on flags
if [ "$INCLUDE_TESTS" = true ]; then
    EXCLUDE_PATTERNS=$(echo "$EXCLUDE_PATTERNS" | sed 's/_test\.go|//g' | sed 's/testdata\/|//g')
fi

if [ "$INCLUDE_GENERATED" = true ]; then
    EXCLUDE_PATTERNS=$(echo "$EXCLUDE_PATTERNS" | sed 's/\.pb\.go|//g' | sed 's/_grpc\.pb\.go|//g' | sed 's/\.connect\.go|//g')
fi

# Get language for extension
get_lang() {
    local ext="${1#.}"
    echo "${LANG_MAP[$ext]:-}"
}

# Header
echo -e "${CYAN}╔═══════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║       Code Exporter                                               ║${NC}"
echo -e "${CYAN}╚═══════════════════════════════════════════════════════════════════╝${NC}"
echo ""

echo -e "${BLUE}[1/3] Initializing...${NC}"
echo -e "  Directories: ${YELLOW}${DIRS}${NC}"
echo -e "  Output: ${YELLOW}${OUTPUT}${NC}"

# Start output file
cat > "$OUTPUT" << EOF
# Logistics Platform - Source Code

> Generated: $(date '+%Y-%m-%d %H:%M:%S')

## Table of Contents

EOF

echo -e "\n${BLUE}[2/3] Collecting files...${NC}"

# Collect files
IFS=',' read -ra DIR_ARRAY <<< "$DIRS"
FILES=()

for dir in "${DIR_ARRAY[@]}"; do
    dir=$(echo "$dir" | xargs)  # Trim whitespace

    if [ ! -d "$dir" ]; then
        echo -e "  ${YELLOW}⚠ Directory not found: ${dir}${NC}"
        continue
    fi

    while IFS= read -r -d '' file; do
        # Check exclude patterns
        if echo "$file" | grep -qE "$EXCLUDE_PATTERNS"; then
            continue
        fi

        # Check extension
        ext="${file##*.}"
        if [ -z "${LANG_MAP[$ext]:-}" ]; then
            continue
        fi

        FILES+=("$file")
    done < <(find "$dir" -type f \( -name "*.go" -o -name "*.proto" -o -name "*.sql" -o -name "*.yaml" -o -name "*.yml" -o -name "*.json" -o -name "*.toml" -o -name "*.sh" -o -name "*.md" \) -print0 2>/dev/null | sort -z)
done

TOTAL_FILES=${#FILES[@]}
echo -e "  Found ${GREEN}${TOTAL_FILES}${NC} files"

# Generate TOC
CURRENT_DIR=""
for file in "${FILES[@]}"; do
    dir=$(dirname "$file")
    if [ "$dir" != "$CURRENT_DIR" ]; then
        CURRENT_DIR="$dir"
        anchor=$(echo "$dir" | tr '/' '-' | tr '.' '' | tr '[:upper:]' '[:lower:]')
        echo "- [$dir](#$anchor)" >> "$OUTPUT"
    fi
done

cat >> "$OUTPUT" << EOF

---

## Statistics

| Metric | Value |
|--------|-------|
| Total Files | $TOTAL_FILES |
| Generated | $(date '+%Y-%m-%d %H:%M:%S') |

---

## Source Files

EOF

echo -e "\n${BLUE}[3/3] Exporting to ${OUTPUT}...${NC}"

# Export files
CURRENT_DIR=""
for file in "${FILES[@]}"; do
    dir=$(dirname "$file")
    filename=$(basename "$file")
    ext="${file##*.}"
    lang=$(get_lang "$ext")
    lines=$(wc -l < "$file" 2>/dev/null || echo "0")
    TOTAL_LINES=$((TOTAL_LINES + lines))

    # Directory header
    if [ "$dir" != "$CURRENT_DIR" ]; then
        CURRENT_DIR="$dir"
        echo "" >> "$OUTPUT"
        echo "### $dir" >> "$OUTPUT"
        echo "" >> "$OUTPUT"
    fi

    # File header
    cat >> "$OUTPUT" << EOF
#### \`$filename\`

> Path: \`$file\` | Lines: $lines

\`\`\`$lang
EOF

    # File content
    cat "$file" >> "$OUTPUT"

    # Ensure newline at end
    if [ -n "$(tail -c 1 "$file")" ]; then
        echo "" >> "$OUTPUT"
    fi

    echo '```' >> "$OUTPUT"
    echo "" >> "$OUTPUT"
done

# Summary
echo -e "\n${GREEN}=== Export Complete ===${NC}"
echo -e "  Output: ${CYAN}${OUTPUT}${NC}"
echo -e "  Files: ${GREEN}${TOTAL_FILES}${NC}"
echo -e "  Lines: ${GREEN}${TOTAL_LINES}${NC}"

FILE_SIZE=$(ls -lh "$OUTPUT" | awk '{print $5}')
echo -e "  Size: ${GREEN}${FILE_SIZE}${NC}"

echo ""
echo -e "${CYAN}╔═══════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║       Done!                                                       ║${NC}"
echo -e "${CYAN}╚═══════════════════════════════════════════════════════════════════╝${NC}"
