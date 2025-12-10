# Flowly

**Network Flow Optimization Platform**

Платформа для оптимизации сетевых потоков в логистике. Решает задачи максимального потока, минимальной стоимости и анализа узких мест в транспортных сетях.

## Возможности

- Оптимизация потоков — алгоритмы Edmonds-Karp, Dinic, Push-Relabel, Min-Cost Flow
- Аналитика — анализ узких мест, статистика, прогнозирование
- Симуляции — Monte Carlo, анализ чувствительности, стресс-тесты
- Отчёты — генерация в PDF, Excel, JSON, Markdown
- Аутентификация — JWT, управление пользователями
- Мониторинг — Prometheus метрики, Jaeger трейсинг

## Быстрый старт

Клонировать репозиторий:

    git clone https://github.com/your-org/flowly.git
    cd flowly

Запустить в dev-режиме:

    make dev

Или только инфраструктуру:

    make infra

API доступен на http://localhost:8080

## Сервисы

| Сервис | Порт | Описание |
|--------|------|----------|
| gateway-svc | 8080 | API Gateway (HTTP/ConnectRPC) |
| solver-svc | 50054 | Алгоритмы оптимизации |
| validation-svc | 50052 | Валидация графов |
| analytics-svc | 50053 | Аналитика и статистика |
| auth-svc | 50055 | Аутентификация |
| history-svc | 50056 | История расчётов |
| audit-svc | 50057 | Аудит действий |
| simulation-svc | 50058 | Симуляции |
| report-svc | 50059 | Генерация отчётов |

## Команды

    make help           # Справка
    make dev            # Dev окружение с hot-reload
    make build          # Сборка всех сервисов
    make test           # Unit тесты
    make lint           # Линтер
    make docker-build   # Сборка Docker образов

## Технологии

- Go 1.25
- gRPC / ConnectRPC
- PostgreSQL
- Redis
- Prometheus + Grafana
- Jaeger
- Kubernetes + Helm

## Лицензия

MIT
