# Dependency Analyzer

A tool for recursively analyzing internal dependencies of Go services. It generates statistics, a dependency tree, and ready-to-use `COPY` commands for Dockerfiles.

## 🎯 Features

-   **Recursive Analysis** — BFS traversal of all transitive dependencies
-   **Multiple Service Support** — Analyze one, several, or all services
-   **Categorization** — Automatic grouping by `gen/`, `pkg/`, `services/`, `migrations/`
-   **Dependency Tree** — Visualization of the dependency graph
-   **Dockerfile COPY** — Ready-made commands for multi-stage builds
-   **Global Statistics** — A summary table for all services
-   **Cross-platform** — Go, Bash, PowerShell

## 📋 Requirements

-   Go 1.25+
-   The project must be in `$GOPATH` or use Go modules

## 🚀 Quick Start

### Go (Recommended)

```/dev/null/bash#L1-6
# Analyze all services
go run scripts/deps/main.go

# Specific service
go run scripts/deps/main.go -path "./services/simulation-svc/..."

# With dependency tree
go run scripts/deps/main.go -tree -depth 3
```

### Bash (Linux/macOS)

```/dev/null/bash#L1-3
chmod +x scripts/deps/list-deps.sh
./scripts/deps/list-deps.sh
```

### PowerShell (Windows)

```/dev/null/powershell#L1-4
# First run — allow script execution
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser

.\scripts\deps\list-deps.ps1
```

## 📖 Usage

### Flags (Go Version)

| Flag        | Description                               | Default       |
| :---------- | :---------------------------------------- | :------------ |
| `-path`     | Path(s) to services (comma-separated)     | All services  |
| `-tree`     | Show dependency tree                      | `false`       |
| `-depth`    | Tree depth                                | `3`           |
| `-docker`   | Only Dockerfile COPY commands             | `false`       |
| `-all`      | Show full list of dependencies            | `false`       |
| `-detailed` | Detailed statistics per service           | `true`        |

### Flags (Bash Version)

| Flag        | Description                         |
| :---------- | :---------------------------------- |
| `--tree`    | Show dependency tree                |
| `--depth N` | Tree depth (default: 3)             |
| `--docker`  | Only Dockerfile COPY commands       |
| `--help`    | Help                                |

### Flags (PowerShell Version)

| Parameter       | Description                         |
| :------------   | :---------------------------------- |
| `-ServicePaths` | Array of service paths              |
| `-ShowTree`     | Show dependency tree                |
| `-TreeDepth`    | Tree depth (default: 3)             |
| `-DockerOnly`   | Only Dockerfile COPY commands       |
| `-Help`         | Help                                |

## 💡 Examples

### Analyzing a Single Service

```/dev/null/bash#L1-1
go run scripts/deps/main.go -path "./services/auth-svc/..."
```

Output:

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

### Analyzing Multiple Services

```/dev/null/bash#L1-1
go run scripts/deps/main.go -path "./services/auth-svc/...,./services/gateway-svc/..."
```

### Dependency Tree

```/dev/null/bash#L1-1
go run scripts/deps/main.go -path "./services/simulation-svc/..." -tree -depth 2
```

Output:

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

### Dockerfile COPY Only

```/dev/null/bash#L1-1
go run scripts/deps/main.go -path "./services/simulation-svc/..." -docker
```

Output:

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

### Global Statistics

When analyzing all services, a summary table is displayed (in an example not real data):

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

## 🏗️ Dockerfile Integration

Use the output to create an optimized multi-stage Dockerfile:

```/dev/null/Dockerfile#L1-34
# syntax=docker/dockerfile:1
FROM golang:1.25.4-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum
COPY go.mod go.sum ./
RUN go mod download

# Copy only necessary dependencies (script output)
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

## 🔧 How it Works

*   **Package Discovery** — `go list ./services/xxx/...` finds all service packages
*   **BFS Traversal** — For each package, imports are extracted using `go list -f '{{.Imports}}'`
*   **Filtering** — Only internal imports (`logistics/...`) are kept
*   **Categorization** — Grouping by prefixes (`gen/`, `pkg/`, `services/`, `migrations/`)
*   **Generation** — `COPY` commands are formed with the required path depth

## 📁 File Structure

```/dev/null/text#L1-6
scripts/deps/
├── main.go           # Go version (recommended)
├── list-deps.sh      # Bash version (Linux/macOS)
├── list-deps.ps1     # PowerShell version (Windows)
└── README.en.md      # This documentation
```

## ⚠️ Known Limitations

*   Analyzes only internal dependencies (within the `logistics` module)
*   Does not account for build tags (`//go:build`)
*   Does not analyze test dependencies (`_test.go`) — only main code
*   Circular dependencies are noted but do not block the analysis

## 🐛 Troubleshooting

### "No packages found"

Ensure that:

*   You are in the project root
*   The path is correct (with `/...` suffix for recursion)
*   `go.mod` exists and is valid

```/dev/null/bash#L1-2
# Check
go list ./services/simulation-svc/...
```

### Empty Category Output

Check that dependencies use the correct module:

```/dev/null/bash#L1-2
go list -m
# Should output: logistics
```

### Errors on Windows

Use the PowerShell version or Go version. The Bash script requires WSL or Git Bash.
