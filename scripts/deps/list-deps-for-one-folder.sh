#!/bin/bash
# Рекурсивно находит все внутренние зависимости для сервиса

set -e

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Модуль проекта
MODULE_NAME="logistics"

# Сервис по умолчанию
SERVICE_PATH="${1:-./services/simulation-svc/...}"

# Временные файлы
VISITED_FILE=$(mktemp)
QUEUE_FILE=$(mktemp)
ALL_DEPS_FILE=$(mktemp)

# Очистка при выходе
cleanup() {
    rm -f "$VISITED_FILE" "$QUEUE_FILE" "$ALL_DEPS_FILE"
}
trap cleanup EXIT

# Функция получения импортов для пакета
get_imports() {
    local pkg="$1"
    go list -f '{{range .Imports}}{{.}}{{"\n"}}{{end}}' "$pkg" 2>/dev/null | \
        grep "^${MODULE_NAME}/" | \
        sort -u
}

echo -e "${CYAN}╔═══════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║     Recursive Dependency Analyzer                 ║${NC}"
echo -e "${CYAN}╚═══════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${YELLOW}Analyzing:${NC} $SERVICE_PATH"
echo -e "${YELLOW}Module:${NC} $MODULE_NAME"
echo ""

# Получаем начальные пакеты
echo -e "${BLUE}[1/4] Finding initial packages...${NC}"
INITIAL_PKGS=$(go list "$SERVICE_PATH" 2>/dev/null)

if [ -z "$INITIAL_PKGS" ]; then
    echo -e "${RED}Error: No packages found at $SERVICE_PATH${NC}"
    exit 1
fi

echo "$INITIAL_PKGS" | while read pkg; do
    echo "  - $pkg"
done

# Инициализируем очередь начальными пакетами
echo "$INITIAL_PKGS" > "$QUEUE_FILE"

# BFS по зависимостям
echo ""
echo -e "${BLUE}[2/4] Resolving transitive dependencies...${NC}"

iteration=0
while [ -s "$QUEUE_FILE" ]; do
    iteration=$((iteration + 1))

    # Читаем текущую очередь
    CURRENT_QUEUE=$(cat "$QUEUE_FILE")
    > "$QUEUE_FILE"  # Очищаем очередь

    new_deps=0

    for pkg in $CURRENT_QUEUE; do
        # Пропускаем если уже посещали
        if grep -Fxq "$pkg" "$VISITED_FILE" 2>/dev/null; then
            continue
        fi

        # Отмечаем как посещённый
        echo "$pkg" >> "$VISITED_FILE"
        echo "$pkg" >> "$ALL_DEPS_FILE"

        # Получаем зависимости этого пакета
        DEPS=$(get_imports "$pkg")

        for dep in $DEPS; do
            # Добавляем в очередь если ещё не посещали
            if ! grep -Fxq "$dep" "$VISITED_FILE" 2>/dev/null; then
                echo "$dep" >> "$QUEUE_FILE"
                new_deps=$((new_deps + 1))
            fi
        done
    done

    if [ $new_deps -gt 0 ]; then
        echo "  Iteration $iteration: found $new_deps new dependencies"
    fi
done

# Сортируем и убираем дубликаты
sort -u "$ALL_DEPS_FILE" -o "$ALL_DEPS_FILE"

TOTAL_DEPS=$(wc -l < "$ALL_DEPS_FILE" | tr -d ' ')

echo ""
echo -e "${BLUE}[3/4] Analyzing results...${NC}"
echo -e "  Total internal packages: ${GREEN}$TOTAL_DEPS${NC}"

# Группируем по директориям
echo ""
echo -e "${BLUE}[4/4] Generating report...${NC}"
echo ""

echo -e "${GREEN}=== All Internal Dependencies (Full Paths) ===${NC}"
cat "$ALL_DEPS_FILE" | sed "s|^${MODULE_NAME}/||"

echo ""
echo -e "${GREEN}=== Required Top-Level Directories ===${NC}"
TOP_DIRS=$(cat "$ALL_DEPS_FILE" | sed "s|^${MODULE_NAME}/||" | cut -d'/' -f1 | sort -u)
for dir in $TOP_DIRS; do
    count=$(cat "$ALL_DEPS_FILE" | grep "^${MODULE_NAME}/${dir}/" | wc -l | tr -d ' ')
    echo "  $dir/ ($count packages)"
done

echo ""
echo -e "${GREEN}=== COPY Commands for Dockerfile ===${NC}"
echo ""

# Собираем директории по категориям
GEN_DIRS=$(cat "$ALL_DEPS_FILE" | grep "^${MODULE_NAME}/gen/" | sed "s|^${MODULE_NAME}/||" | cut -d'/' -f1-4 | sort -u)
PKG_DIRS=$(cat "$ALL_DEPS_FILE" | grep "^${MODULE_NAME}/pkg/" | sed "s|^${MODULE_NAME}/||" | cut -d'/' -f1-2 | sort -u)
SVC_DIRS=$(cat "$ALL_DEPS_FILE" | grep "^${MODULE_NAME}/services/" | sed "s|^${MODULE_NAME}/||" | cut -d'/' -f1-2 | sort -u)
MIGRATIONS_DIRS=$(cat "$ALL_DEPS_FILE" | grep "^${MODULE_NAME}/migrations" | sed "s|^${MODULE_NAME}/||" | cut -d'/' -f1 | sort -u)

# Находим "Другое" - всё что не попало в известные категории
OTHER_DEPS=$(cat "$ALL_DEPS_FILE" | sed "s|^${MODULE_NAME}/||" | \
    grep -v "^gen/" | \
    grep -v "^pkg/" | \
    grep -v "^services/" | \
    grep -v "^migrations" | \
    sort -u)

# Выводим COPY команды
if [ -n "$GEN_DIRS" ]; then
    echo "# Generated proto files"
    echo "$GEN_DIRS" | while read dir; do
        echo "COPY ${dir}/ ./${dir}/"
    done
    echo ""
fi

if [ -n "$PKG_DIRS" ]; then
    echo "# Shared packages"
    echo "$PKG_DIRS" | while read dir; do
        echo "COPY ${dir}/ ./${dir}/"
    done
    echo ""
fi

if [ -n "$SVC_DIRS" ]; then
    echo "# Services"
    echo "$SVC_DIRS" | while read dir; do
        echo "COPY ${dir}/ ./${dir}/"
    done
    echo ""
fi

if [ -n "$MIGRATIONS_DIRS" ]; then
    echo "# Migrations"
    echo "$MIGRATIONS_DIRS" | while read dir; do
        echo "COPY ${dir}/ ./${dir}/"
    done
    echo ""
fi

if [ -n "$OTHER_DEPS" ]; then
    echo -e "${YELLOW}# Other (not categorized)${NC}"
    # Группируем по первым двум уровням директорий
    OTHER_DIRS=$(echo "$OTHER_DEPS" | cut -d'/' -f1-2 | sort -u)
    echo "$OTHER_DIRS" | while read dir; do
        echo "COPY ${dir}/ ./${dir}/"
    done
    echo ""

    echo -e "${YELLOW}# Detailed list of 'Other' dependencies:${NC}"
    echo "$OTHER_DEPS" | while read dep; do
        echo "#   - $dep"
    done
    echo ""
fi

# Проверка на отсутствующие зависимости
echo -e "${GREEN}=== Summary ===${NC}"
echo ""
echo "Categories found:"
[ -n "$GEN_DIRS" ] && echo "  ✓ gen/ (proto files)"
[ -n "$PKG_DIRS" ] && echo "  ✓ pkg/ (shared packages)"
[ -n "$SVC_DIRS" ] && echo "  ✓ services/"
[ -n "$MIGRATIONS_DIRS" ] && echo "  ✓ migrations/"
if [ -n "$OTHER_DEPS" ]; then
    echo -e "  ${YELLOW}⚠ Other (review required)${NC}"
    OTHER_COUNT=$(echo "$OTHER_DEPS" | wc -l | tr -d ' ')
    echo -e "    ${YELLOW}Found $OTHER_COUNT uncategorized dependencies${NC}"
fi

echo ""
echo -e "${CYAN}╔═══════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║     Analysis Complete                             ║${NC}"
echo -e "${CYAN}╚═══════════════════════════════════════════════════╝${NC}"
