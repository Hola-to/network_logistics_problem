# Dependency Analyzer

Инструмент для рекурсивного анализа внутренних зависимостей Go-сервисов. Генерирует статистику, дерево зависимостей и готовые команды `COPY` для Dockerfile.

## 🎯 Возможности

- **Рекурсивный анализ** — BFS-обход всех транзитивных зависимостей
- **Поддержка нескольких сервисов** — анализ одного, нескольких или всех сервисов
- **Категоризация** — автоматическая группировка по `gen/`, `pkg/`, `services/`, `migrations/`
- **Дерево зависимостей** — визуализация графа зависимостей
- **Dockerfile COPY** — готовые команды для многоступенчатой сборки
- **Глобальная статистика** — сводная таблица по всем сервисам
- **Кросс-платформенность** — Go, Bash, PowerShell

## 📋 Требования

- Go 1.25+
- Проект должен быть в `$GOPATH` или использовать Go modules

## 🚀 Быстрый старт

### Go (рекомендуется)

```/dev/null/bash#L1-6
# Анализ всех сервисов
go run scripts/deps/main.go

# Конкретный сервис
go run scripts/deps/main.go -path "./services/simulation-svc/..."

# С деревом зависимостей
go run scripts/deps/main.go -tree -depth 3
```

### Bash (Linux/macOS)

```/dev/null/bash#L1-3
chmod +x scripts/deps/list-deps.sh
./scripts/deps/list-deps.sh
```

### PowerShell (Windows)

```/dev/null/powershell#L1-4
# Первый запуск — разрешить выполнение скриптов
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser

.\scripts\deps\list-deps.ps1
```

## 📖 Использование

### Флаги (Go версия)

| Флаг        | Описание                            | По умолчанию  |
| :---------- | :---------------------------------- | :------------ |
| `-path`     | Путь(и) к сервисам (через запятую)  | Все сервисы   |
| `-tree`     | Показать дерево зависимостей        | `false`       |
| `-depth`    | Глубина дерева                      | `3`           |
| `-docker`   | Только Dockerfile COPY команды      | `false`       |
| `-all`      | Показать полный список зависимостей | `false`       |
| `-detailed` | Детальная статистика по сервисам    | `true`        |

### Флаги (Bash версия)

| Флаг       | Описание                            |
| :--------- | :---------------------------------- |
| `--tree`   | Показать дерево зависимостей        |
| `--depth N`| Глубина дерева (по умолчанию: 3)    |
| `--docker` | Только Dockerfile COPY команды      |
| `--help`   | Справка                             |

### Флаги (PowerShell версия)

| Параметр       | Описание                            |
| :------------- | :---------------------------------- |
| `-ServicePaths`| Массив путей к сервисам             |
| `-ShowTree`    | Показать дерево зависимостей        |
| `-TreeDepth`   | Глубина дерева (по умолчанию: 3)    |
| `-DockerOnly`  | Только Dockerfile COPY команды      |
| `-Help`        | Справка                             |

## 💡 Примеры

### Анализ одного сервиса

```/dev/null/bash#L1-1
go run scripts/deps/main.go -path "./services/auth-svc/..."
```

Вывод:

```/dev/null/text#L1-20
╔═══════════════════════════════════════════════════════════════════╗
║       Recursive Dependency Analyzer v2.0                          ║
╚═══════════════════════════════════════════════════════════════════╝

[1/4] Finding initial packages...
Analyzing: ./services/auth-svc/...

[2/4] Resolving transitive dependencies...
  Iteration 1: found 12 new dependencies
  Iteration 2: found 5 new dependencies
  Iteration 3: found 2 new dependencies

━━━ auth-svc ━━━
  Path: ./services/auth-svc/...
  Total packages: 24

  Categories:
    ✓ Generated proto files    4 directories
    ✓ Shared packages          8 directories
    ✓ Services                 1 directories
    ✓ Migrations               1 directories
```

### Анализ нескольких сервисов

```/dev/null/bash#L1-1
go run scripts/deps/main.go -path "./services/auth-svc/...,./services/gateway-svc/..."
```

### Дерево зависимостей

```/dev/null/bash#L1-1
go run scripts/deps/main.go -path "./services/simulation-svc/..." -tree -depth 2
```

Вывод:

```/dev/null/text#L1-14
=== Dependency Tree (max depth: 2) ===

├── services/simulation-svc/cmd
│   ├── services/simulation-svc/internal/service
│   │   ├── services/simulation-svc/internal/engine
│   │   ├── services/simulation-svc/internal/repository
│   │   └── gen/go/logistics/simulation/v1
│   ├── pkg/config
│   │   ├── pkg/logger
│   │   └── pkg/telemetry
│   └── pkg/server
│       ├── pkg/interceptors
│       └── pkg/metrics
```

### Только Dockerfile COPY

```/dev/null/bash#L1-1
go run scripts/deps/main.go -path "./services/simulation-svc/..." -docker
```

Вывод:

```/dev/null/text#L1-19
=== Dockerfile COPY Commands ===

# Generated proto files
COPY gen/go/logistics/common/v1/ ./gen/go/logistics/common/v1/
COPY gen/go/logistics/simulation/v1/ ./gen/go/logistics/simulation/v1/
COPY gen/go/logistics/solver/v1/ ./gen/go/logistics/solver/v1/

# Shared packages
COPY pkg/cache/ ./pkg/cache/
COPY pkg/client/ ./pkg/client/
COPY pkg/config/ ./pkg/config/
COPY pkg/database/ ./pkg/database/
COPY pkg/logger/ ./pkg/logger/

# Services
COPY services/simulation-svc/ ./services/simulation-svc/

# Migrations
COPY migrations/ ./migrations/
```

### Глобальная статистика

При анализе всех сервисов выводится сводная таблица (в примере не реальные данные):

```/dev/null/text#L1-20
╔═══════════════════════════════════════════════════════════════════╗
║       Global Statistics                                           ║
╚═══════════════════════════════════════════════════════════════════╝

Service                       Total       gen/       pkg/  services/      Other
────────────────────────────────────────────────────────────────────────────────
analytics-svc                    18          3          6          2          0
audit-svc                        15          2          5          1          0
auth-svc                         24          4          8          1          0
gateway-svc                      42          8         12          6          0
history-svc                      16          3          5          1          0
report-svc                       28          4          9          2          0
simulation-svc                   35          5         11          3          0
solver-svc                       22          3          7          1          0
validation-svc                   19          3          6          1          0
────────────────────────────────────────────────────────────────────────────────
TOTAL (with duplicates)         219
```

## 🏗️ Интеграция с Dockerfile

Используйте вывод для создания оптимизированного многоступенчатого Dockerfile:

```/dev/null/Dockerfile#L1-34
# syntax=docker/dockerfile:1
FROM golang:1.25.4-alpine AS builder

WORKDIR /app

# Копируем go.mod и go.sum
COPY go.mod go.sum ./
RUN go mod download

# Копируем только необходимые зависимости (вывод скрипта)
# Generated proto files
COPY gen/go/logistics/common/v1/ ./gen/go/logistics/common/v1/
COPY gen/go/logistics/simulation/v1/ ./gen/go/logistics/simulation/v1/

# Shared packages
COPY pkg/cache/ ./pkg/cache/
COPY pkg/config/ ./pkg/config/
COPY pkg/database/ ./pkg/database/
COPY pkg/logger/ ./pkg/logger/

# Service
COPY services/simulation-svc/ ./services/simulation-svc/

# Migrations
COPY migrations/ ./migrations/

# Build
RUN go build -o /bin/service ./services/simulation-svc/cmd

FROM alpine:3.19
COPY --from=builder /bin/service /bin/service
ENTRYPOINT ["/bin/service"]
```

## 🔧 Как это работает

*   **Поиск пакетов** — `go list ./services/xxx/...` находит все пакеты сервиса
*   **BFS-обход** — для каждого пакета извлекаются импорты через `go list -f '{{.Imports}}'`
*   **Фильтрация** — оставляем только внутренние импорты (`logistics/...`)
*   **Категоризация** — группируем по префиксам (`gen/`, `pkg/`, `services/`, `migrations/`)
*   **Генерация** — формируем `COPY` команды с нужной глубиной пути

## 📁 Структура файлов

```/dev/null/text#L1-6
scripts/deps/
├── main.go           # Go версия (рекомендуется)
├── list-deps.sh      # Bash версия (Linux/macOS)
├── list-deps.ps1     # PowerShell версия (Windows)
└── README.md         # Эта документация
```

## ⚠️ Известные ограничения

*   Анализирует только внутренние зависимости (внутри модуля `logistics`)
*   Не учитывает build tags (`//go:build`)
*   Не анализирует зависимости тестов (`_test.go`) — только основной код
*   Циклические зависимости отмечаются, но не блокируют анализ

## 🐛 Troubleshooting

### "No packages found"

Убедитесь, что:

*   Вы находитесь в корне проекта
*   Путь корректный (с суффиксом `/...` для рекурсии)
*   `go.mod` существует и валиден

```/dev/null/bash#L1-2
# Проверка
go list ./services/simulation-svc/...
```

### Пустой вывод категорий

Проверьте, что зависимости используют правильный модуль:

```/dev/null/bash#L1-2
go list -m
# Должно вывести: logistics
```

### Ошибки на Windows

Используйте PowerShell версию или Go версию. Bash-скрипт требует WSL или Git Bash.
