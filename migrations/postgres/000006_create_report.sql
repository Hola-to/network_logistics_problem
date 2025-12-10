-- +goose Up
CREATE TABLE IF NOT EXISTS reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    description TEXT,
    author VARCHAR(255),
    report_type VARCHAR(50) NOT NULL,
    format VARCHAR(20) NOT NULL,
    content BYTEA NOT NULL,
    content_type VARCHAR(100) NOT NULL,
    filename VARCHAR(255) NOT NULL,
    size_bytes BIGINT NOT NULL,
    calculation_id VARCHAR(36),
    graph_id VARCHAR(36),
    user_id VARCHAR(36),
    generation_time_ms DOUBLE PRECISION,
    version VARCHAR(50),
    tags TEXT[] DEFAULT '{}',
    custom_fields JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_reports_created_at ON reports(created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_reports_report_type ON reports(report_type) WHERE deleted_at IS NULL;
CREATE INDEX idx_reports_format ON reports(format) WHERE deleted_at IS NULL;
CREATE INDEX idx_reports_user_id ON reports(user_id) WHERE deleted_at IS NULL AND user_id IS NOT NULL;
CREATE INDEX idx_reports_calculation_id ON reports(calculation_id) WHERE deleted_at IS NULL AND calculation_id IS NOT NULL;
CREATE INDEX idx_reports_graph_id ON reports(graph_id) WHERE deleted_at IS NULL AND graph_id IS NOT NULL;
CREATE INDEX idx_reports_tags ON reports USING GIN(tags) WHERE deleted_at IS NULL;
CREATE INDEX idx_reports_expires_at ON reports(expires_at) WHERE deleted_at IS NULL AND expires_at IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS reports;
