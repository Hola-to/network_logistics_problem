-- +goose Up
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    service VARCHAR(100) NOT NULL,
    method VARCHAR(255) NOT NULL,
    request_id VARCHAR(100),
    action VARCHAR(50) NOT NULL,
    outcome VARCHAR(50) NOT NULL,
    user_id VARCHAR(255),
    username VARCHAR(255),
    user_role VARCHAR(50),
    client_ip INET,
    user_agent TEXT,
    resource_type VARCHAR(100),
    resource_id VARCHAR(255),
    resource_name VARCHAR(255),
    duration_ms BIGINT,
    error_code VARCHAR(100),
    error_message TEXT,
    changes_before JSONB,
    changes_after JSONB,
    changed_fields TEXT[],
    metadata JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_audit_logs_timestamp ON audit_logs(timestamp DESC);
CREATE INDEX idx_audit_logs_service ON audit_logs(service);
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_outcome ON audit_logs(outcome);
CREATE INDEX idx_audit_logs_user_time ON audit_logs(user_id, timestamp DESC);
CREATE INDEX idx_audit_logs_resource_time ON audit_logs(resource_type, resource_id, timestamp DESC);
CREATE INDEX idx_audit_logs_service_time ON audit_logs(service, timestamp DESC);
CREATE INDEX idx_audit_logs_metadata ON audit_logs USING GIN(metadata);

-- +goose Down
DROP TABLE IF EXISTS audit_logs;
