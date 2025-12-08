#!/bin/bash
# Рекурсивно находит все внутренние зависимости для сервиса(ов)

set -e

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
GRAY='\033[0;90m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Модуль проекта
MODULE_NAME="logistics"

# Аргументы
SERVICE_PATHS=("$@")
SHOW_TREE=false
TREE_DEPTH=3
DOCKER_ONLY=false

# Парсинг аргументов
while [[ $# -gt 0 ]]; do
    case $1 in
        --tree)
            SHOW_TREE=true
            shift
            ;;
        --depth)
            TREE_DEPTH="$2"
            shift 2
            ;;
        --docker)
            DOCKER_ONLY=true
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [OPTIONS] [SERVICE_PATHS...]"
            echo ""
            echo "Options:"
            echo "  --tree          Show dependency tree"
            echo "  --depth N       Tree depth (default: 3)"
            echo "  --docker        Show only Dockerfile COPY commands"
            echo "  --help, -h      Show this help"
            echo ""
            echo "Examples:"
            echo "  $0                                    # Analyze all services"
            echo "  $0 ./services/simulation-svc/...     # Analyze single service"
            echo "  $0 --tree --depth 2 ./services/auth-svc/..."
            exit 0
            ;;
        *)
            # Assume it's a service path if not a flag
            if [[ ! "$1" =~ ^-- ]]; then
                SERVICE_PATHS+=("$1")
            fi
            shift
            ;;
    esac
done

# Временные файлы
VISITED_FILE=$(mktemp)
QUEUE_FILE=$(mktemp)
ALL_DEPS_FILE=$(mktemp)
DEP_GRAPH_FILE=$(mktemp)
STATS_FILE=$(mktemp)

# Очистка при выходе
cleanup() {
    rm -f "$VISITED_FILE" "$QUEUE_FILE" "$ALL_DEPS_FILE" "$DEP_GRAPH_FILE" "$STATS_FILE"
}
trap cleanup EXIT

# Функция получения импортов для пакета
get_imports() {
    local pkg="$1"
    go list -f '{{range .Imports}}{{.}}{{"\n"}}{{end}}' "$pkg" 2>/dev/null | \
        grep "^${MODULE_NAME}/" | \
        sort -u
}

# Функция для нахождения всех сервисов
find_all_services() {
    find ./services -maxdepth 1 -type d -name "*-svc" | sort | while read -r dir; do
        echo "${dir}/..."
    done
}

# Функция анализа зависимостей
analyze_service() {
    local service_path="$1"
    local service_name=$(basename "${service_path%/...}")

    # Очищаем временные файлы
    > "$VISITED_FILE"
    > "$QUEUE_FILE"
    > "$ALL_DEPS_FILE"
    > "$DEP_GRAPH_FILE"

    echo -e "\n${CYAN}━━━ ${BOLD}${service_name}${NC}${CYAN} ━━━${NC}"
    echo -e "  Path: ${GRAY}${service_path}${NC}"

    # Получаем начальные пакеты
    INITIAL_PKGS=$(go list "$service_path" 2>/dev/null)

    if [ -z "$INITIAL_PKGS" ]; then
        echo -e "  ${RED}Error: No packages found${NC}"
        return 1
    fi

    # Инициализируем очередь
    echo "$INITIAL_PKGS" > "$QUEUE_FILE"

    # BFS
    iteration=0
    while [ -s "$QUEUE_FILE" ]; do
        iteration=$((iteration + 1))
        CURRENT_QUEUE=$(cat "$QUEUE_FILE")
        > "$QUEUE_FILE"

        new_deps=0

        for pkg in $CURRENT_QUEUE; do
            if grep -Fxq "$pkg" "$VISITED_FILE" 2>/dev/null; then
                continue
            fi

            echo "$pkg" >> "$VISITED_FILE"
            echo "$pkg" >> "$ALL_DEPS_FILE"

            DEPS=$(get_imports "$pkg")

            for dep in $DEPS; do
                # Сохраняем граф зависимостей
                echo "$pkg -> $dep" >> "$DEP_GRAPH_FILE"

                if ! grep -Fxq "$dep" "$VISITED_FILE" 2>/dev/null; then
                    echo "$dep" >> "$QUEUE_FILE"
                    new_deps=$((new_deps + 1))
                fi
            done
        done

        if [ $new_deps -gt 0 ]; then
            echo -e "  Iteration $iteration: found ${GREEN}$new_deps${NC} new dependencies"
        fi
    done

    # Подсчёт статистики
    sort -u "$ALL_DEPS_FILE" -o "$ALL_DEPS_FILE"
    TOTAL_DEPS=$(wc -l < "$ALL_DEPS_FILE" | tr -d ' ')

    # Подсчёт по категориям
    GEN_COUNT=$(grep "^${MODULE_NAME}/gen/" "$ALL_DEPS_FILE" 2>/dev/null | wc -l | tr -d ' ')
    PKG_COUNT=$(grep "^${MODULE_NAME}/pkg/" "$ALL_DEPS_FILE" 2>/dev/null | wc -l | tr -d ' ')
    SVC_COUNT=$(grep "^${MODULE_NAME}/services/" "$ALL_DEPS_FILE" 2>/dev/null | wc -l | tr -d ' ')
    MIG_COUNT=$(grep "^${MODULE_NAME}/migrations" "$ALL_DEPS_FILE" 2>/dev/null | wc -l | tr -d ' ')
    OTHER_COUNT=$((TOTAL_DEPS - GEN_COUNT - PKG_COUNT - SVC_COUNT - MIG_COUNT))

    echo -e "  Total packages: ${GREEN}${TOTAL_DEPS}${NC}"
    echo ""
    echo "  Categories:"
    [ $GEN_COUNT -gt 0 ] && echo -e "    ${GREEN}✓${NC} Generated proto files    ${YELLOW}${GEN_COUNT}${NC}"
    [ $PKG_COUNT -gt 0 ] && echo -e "    ${GREEN}✓${NC} Shared packages          ${YELLOW}${PKG_COUNT}${NC}"
    [ $SVC_COUNT -gt 0 ] && echo -e "    ${GREEN}✓${NC} Services                 ${YELLOW}${SVC_COUNT}${NC}"
    [ $MIG_COUNT -gt 0 ] && echo -e "    ${GREEN}✓${NC} Migrations               ${YELLOW}${MIG_COUNT}${NC}"
    [ $OTHER_COUNT -gt 0 ] && echo -e "    ${YELLOW}⚠${NC} Other                    ${RED}${OTHER_COUNT}${NC}"

    # Сохраняем для глобальной статистики
    echo "$service_name,$TOTAL_DEPS,$GEN_COUNT,$PKG_COUNT,$SVC_COUNT,$OTHER_COUNT" >> "$STATS_FILE"

    return 0
}

# Функция вывода дерева зависимостей
print_tree() {
    local pkg="$1"
    local prefix="$2"
    local depth="$3"
    local max_depth="$4"
    local visited_tree="$5"

    if [ "$depth" -gt "$max_depth" ]; then
        return
    fi

    local rel="${pkg#${MODULE_NAME}/}"

    # Проверяем циклические зависимости
    if grep -Fxq "$pkg" "$visited_tree" 2>/dev/null; then
        echo -e "${prefix}└── ${GRAY}${rel} (circular)${NC}"
        return
    fi

    echo "$pkg" >> "$visited_tree"
    echo -e "${prefix}├── ${rel}"

    # Получаем зависимости из графа
    local deps=$(grep "^$pkg -> " "$DEP_GRAPH_FILE" 2>/dev/null | sed "s|^$pkg -> ||" | head -5)

    local count=0
    local total=$(echo "$deps" | wc -w)

    for dep in $deps; do
        count=$((count + 1))
        local new_prefix="${prefix}│   "
        if [ $count -eq $total ]; then
            new_prefix="${prefix}    "
        fi
        print_tree "$dep" "$new_prefix" $((depth + 1)) "$max_depth" "$visited_tree"
    done
}

# Функция вывода COPY команд
print_docker_copy() {
    echo -e "\n${GREEN}=== Dockerfile COPY Commands ===${NC}"

    # Gen
    GEN_DIRS=$(cat "$ALL_DEPS_FILE" | grep "^${MODULE_NAME}/gen/" | sed "s|^${MODULE_NAME}/||" | cut -d'/' -f1-4 | sort -u)
    if [ -n "$GEN_DIRS" ]; then
        echo -e "\n${GRAY}# Generated proto files${NC}"
        echo "$GEN_DIRS" | while read -r dir; do
            echo "COPY ${dir}/ ./${dir}/"
        done
    fi

    # Pkg
    PKG_DIRS=$(cat "$ALL_DEPS_FILE" | grep "^${MODULE_NAME}/pkg/" | sed "s|^${MODULE_NAME}/||" | cut -d'/' -f1-2 | sort -u)
    if [ -n "$PKG_DIRS" ]; then
        echo -e "\n${GRAY}# Shared packages${NC}"
        echo "$PKG_DIRS" | while read -r dir; do
            echo "COPY ${dir}/ ./${dir}/"
        done
    fi

    # Services
    SVC_DIRS=$(cat "$ALL_DEPS_FILE" | grep "^${MODULE_NAME}/services/" | sed "s|^${MODULE_NAME}/||" | cut -d'/' -f1-2 | sort -u)
    if [ -n "$SVC_DIRS" ]; then
        echo -e "\n${GRAY}# Services${NC}"
        echo "$SVC_DIRS" | while read -r dir; do
            echo "COPY ${dir}/ ./${dir}/"
        done
    fi

    # Migrations
    MIG_DIRS=$(cat "$ALL_DEPS_FILE" | grep "^${MODULE_NAME}/migrations" | sed "s|^${MODULE_NAME}/||" | cut -d'/' -f1 | sort -u)
    if [ -n "$MIG_DIRS" ]; then
        echo -e "\n${GRAY}# Migrations${NC}"
        echo "$MIG_DIRS" | while read -r dir; do
            echo "COPY ${dir}/ ./${dir}/"
        done
    fi
}

# Функция вывода глобальной статистики
print_global_stats() {
    echo -e "\n${GREEN}╔═══════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║       Global Statistics                                           ║${NC}"
    echo -e "${GREEN}╚═══════════════════════════════════════════════════════════════════╝${NC}"
    echo ""

    printf "%-25s %10s %10s %10s %10s %10s\n" "Service" "Total" "gen/" "pkg/" "services/" "Other"
    printf "%s\n" "$(printf '─%.0s' {1..80})"

    total_all=0
    while IFS=',' read -r name total gen pkg svc other; do
        printf "%-25s %10s %10s %10s %10s %10s\n" "$name" "$total" "$gen" "$pkg" "$svc" "$other"
        total_all=$((total_all + total))
    done < "$STATS_FILE"

    printf "%s\n" "$(printf '─%.0s' {1..80})"
    printf "%-25s ${BOLD}%10s${NC}\n" "TOTAL (with duplicates)" "$total_all"
}

# === MAIN ===

echo -e "${CYAN}╔═══════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║       Recursive Dependency Analyzer v2.0 (Shell)                  ║${NC}"
echo -e "${CYAN}╚═══════════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Если пути не указаны, находим все сервисы
if [ ${#SERVICE_PATHS[@]} -eq 0 ]; then
    echo -e "${BLUE}[1/4] Discovering services...${NC}"
    mapfile -t SERVICE_PATHS < <(find_all_services)
    echo -e "  Found ${GREEN}${#SERVICE_PATHS[@]}${NC} services"
    for svc in "${SERVICE_PATHS[@]}"; do
        echo -e "    ${CYAN}•${NC} $svc"
    done
fi

echo -e "\n${BLUE}[2/4] Analyzing dependencies...${NC}"

# Анализируем каждый сервис
for svc_path in "${SERVICE_PATHS[@]}"; do
    analyze_service "$svc_path"
done

echo -e "\n${BLUE}[3/4] Generating combined analysis...${NC}"

# Объединённый анализ
> "$VISITED_FILE"
> "$QUEUE_FILE"
> "$ALL_DEPS_FILE"
> "$DEP_GRAPH_FILE"

for svc_path in "${SERVICE_PATHS[@]}"; do
    PKGS=$(go list "$svc_path" 2>/dev/null)
    echo "$PKGS" >> "$QUEUE_FILE"
done

# BFS для объединённого анализа
while [ -s "$QUEUE_FILE" ]; do
    CURRENT_QUEUE=$(cat "$QUEUE_FILE")
    > "$QUEUE_FILE"

    for pkg in $CURRENT_QUEUE; do
        if grep -Fxq "$pkg" "$VISITED_FILE" 2>/dev/null; then
            continue
        fi

        echo "$pkg" >> "$VISITED_FILE"
        echo "$pkg" >> "$ALL_DEPS_FILE"

        DEPS=$(get_imports "$pkg")

        for dep in $DEPS; do
            echo "$pkg -> $dep" >> "$DEP_GRAPH_FILE"
            if ! grep -Fxq "$dep" "$VISITED_FILE" 2>/dev/null; then
                echo "$dep" >> "$QUEUE_FILE"
            fi
        done
    done
done

sort -u "$ALL_DEPS_FILE" -o "$ALL_DEPS_FILE"

echo -e "\n${BLUE}[4/4] Generating report...${NC}"

# Топ-уровневые директории
echo -e "\n${GREEN}=== Required Top-Level Directories ===${NC}"
cat "$ALL_DEPS_FILE" | sed "s|^${MODULE_NAME}/||" | cut -d'/' -f1 | sort | uniq -c | sort -rn | \
    while read -r count dir; do
        printf "  ${BOLD}%-15s${NC} (${YELLOW}%d${NC} packages)\n" "$dir/" "$count"
    done

# COPY команды
if [ "$DOCKER_ONLY" = true ]; then
    print_docker_copy
else
    print_docker_copy

    # Summary
    echo -e "\n${GREEN}=== Summary ===${NC}"
    echo ""
    echo "Categories found:"
    [ $(grep "^${MODULE_NAME}/gen/" "$ALL_DEPS_FILE" 2>/dev/null | wc -l) -gt 0 ] && \
        echo -e "  ${GREEN}✓${NC} Generated proto files"
    [ $(grep "^${MODULE_NAME}/pkg/" "$ALL_DEPS_FILE" 2>/dev/null | wc -l) -gt 0 ] && \
        echo -e "  ${GREEN}✓${NC} Shared packages"
    [ $(grep "^${MODULE_NAME}/services/" "$ALL_DEPS_FILE" 2>/dev/null | wc -l) -gt 0 ] && \
        echo -e "  ${GREEN}✓${NC} Services"
    [ $(grep "^${MODULE_NAME}/migrations" "$ALL_DEPS_FILE" 2>/dev/null | wc -l) -gt 0 ] && \
        echo -e "  ${GREEN}✓${NC} Migrations"
fi

# Дерево зависимостей
if [ "$SHOW_TREE" = true ]; then
    echo -e "\n${GREEN}=== Dependency Tree (max depth: ${TREE_DEPTH}) ===${NC}"

    TREE_VISITED=$(mktemp)
    trap "rm -f $TREE_VISITED" EXIT

    INITIAL_PKGS=$(head -5 "$ALL_DEPS_FILE")
    for pkg in $INITIAL_PKGS; do
        echo ""
        > "$TREE_VISITED"
        print_tree "$pkg" "" 0 "$TREE_DEPTH" "$TREE_VISITED"
    done
fi

# Глобальная статистика
print_global_stats

echo ""
echo -e "${CYAN}╔═══════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║       Analysis Complete                                           ║${NC}"
echo -e "${CYAN}╚═══════════════════════════════════════════════════════════════════╝${NC}"
