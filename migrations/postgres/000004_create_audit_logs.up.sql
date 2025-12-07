-- Таблица аудит логов
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Источник
    service VARCHAR(100) NOT NULL,
    method VARCHAR(255) NOT NULL,
    request_id VARCHAR(100),

    -- Действие
    action VARCHAR(50) NOT NULL,
    outcome VARCHAR(50) NOT NULL,

    -- Актор
    user_id VARCHAR(255),
    username VARCHAR(255),
    user_role VARCHAR(50),

    -- Клиент
    client_ip INET,
    user_agent TEXT,

    -- Ресурс
    resource_type VARCHAR(100),
    resource_id VARCHAR(255),
    resource_name VARCHAR(255),

    -- Детали
    duration_ms BIGINT,
    error_code VARCHAR(100),
    error_message TEXT,

    -- Изменения
    changes_before JSONB,
    changes_after JSONB,
    changed_fields TEXT[],

    -- Метаданные
    metadata JSONB NOT NULL DEFAULT '{}'
);

-- Индексы для частых запросов
CREATE INDEX idx_audit_logs_timestamp ON audit_logs(timestamp DESC);
CREATE INDEX idx_audit_logs_service ON audit_logs(service);
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_outcome ON audit_logs(outcome);

-- Составные индексы
CREATE INDEX idx_audit_logs_user_time ON audit_logs(user_id, timestamp DESC);
CREATE INDEX idx_audit_logs_resource_time ON audit_logs(resource_type, resource_id, timestamp DESC);
CREATE INDEX idx_audit_logs_service_time ON audit_logs(service, timestamp DESC);

-- GIN индекс для поиска по метаданным
CREATE INDEX idx_audit_logs_metadata ON audit_logs USING GIN(metadata);

-- Полнотекстовый поиск
CREATE INDEX idx_audit_logs_search ON audit_logs USING GIN(
    to_tsvector('russian', COALESCE(method, '') || ' ' || COALESCE(error_message, '') || ' ' || COALESCE(resource_name, ''))
);
