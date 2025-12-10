-- +goose Up
CREATE TABLE IF NOT EXISTS calculations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL DEFAULT '',
    algorithm VARCHAR(50) NOT NULL,
    max_flow DOUBLE PRECISION NOT NULL DEFAULT 0,
    total_cost DOUBLE PRECISION NOT NULL DEFAULT 0,
    computation_time_ms DOUBLE PRECISION NOT NULL DEFAULT 0,
    node_count INTEGER NOT NULL DEFAULT 0,
    edge_count INTEGER NOT NULL DEFAULT 0,
    request_data JSONB NOT NULL,
    response_data JSONB NOT NULL,
    tags TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_calculations_user_id ON calculations(user_id);
CREATE INDEX IF NOT EXISTS idx_calculations_algorithm ON calculations(algorithm);
CREATE INDEX IF NOT EXISTS idx_calculations_created_at ON calculations(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_calculations_max_flow ON calculations(max_flow);
CREATE INDEX IF NOT EXISTS idx_calculations_tags ON calculations USING GIN(tags);
CREATE INDEX IF NOT EXISTS idx_calculations_user_created ON calculations(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_calculations_name_search ON calculations USING GIN(to_tsvector('english', name));

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_calculations_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trigger_calculations_updated_at
    BEFORE UPDATE ON calculations
    FOR EACH ROW
    EXECUTE FUNCTION update_calculations_updated_at();
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER IF EXISTS trigger_calculations_updated_at ON calculations;
DROP FUNCTION IF EXISTS update_calculations_updated_at();
DROP TABLE IF EXISTS calculations;
