-- Таблица симуляций
CREATE TABLE IF NOT EXISTS simulations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,

    -- Тип симуляции
    simulation_type VARCHAR(50) NOT NULL,

    -- Метрики для быстрого поиска
    node_count INTEGER NOT NULL DEFAULT 0,
    edge_count INTEGER NOT NULL DEFAULT 0,
    computation_time_ms DOUBLE PRECISION NOT NULL DEFAULT 0,

    -- Ключевые результаты
    baseline_flow DOUBLE PRECISION,
    result_flow DOUBLE PRECISION,
    flow_change_percent DOUBLE PRECISION,

    -- Полные данные (JSONB)
    graph_data JSONB,
    request_data JSONB NOT NULL,
    response_data JSONB NOT NULL,

    -- Теги
    tags TEXT[] NOT NULL DEFAULT '{}',

    -- Временные метки
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Индексы
CREATE INDEX idx_simulations_user_id ON simulations(user_id);
CREATE INDEX idx_simulations_type ON simulations(simulation_type);
CREATE INDEX idx_simulations_created_at ON simulations(created_at DESC);
CREATE INDEX idx_simulations_user_type ON simulations(user_id, simulation_type);
CREATE INDEX idx_simulations_user_created ON simulations(user_id, created_at DESC);
CREATE INDEX idx_simulations_tags ON simulations USING GIN(tags);

-- Полнотекстовый поиск
CREATE INDEX idx_simulations_name_search ON simulations USING GIN(to_tsvector('russian', name || ' ' || COALESCE(description, '')));

-- Триггер для updated_at
CREATE OR REPLACE FUNCTION update_simulations_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_simulations_updated_at
    BEFORE UPDATE ON simulations
    FOR EACH ROW
    EXECUTE FUNCTION update_simulations_updated_at();
