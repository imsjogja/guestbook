-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS event_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ,
    location_id UUID,
    capacity INTEGER,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_event_sessions_event ON event_sessions(event_id);
CREATE INDEX IF NOT EXISTS idx_event_sessions_time ON event_sessions(start_time, end_time);
CREATE INDEX IF NOT EXISTS idx_event_sessions_sort ON event_sessions(event_id, sort_order);
-- +goose StatementEnd
