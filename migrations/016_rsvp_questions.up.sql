-- +goose Up
-- +goose StatementBegin

-- rsvp_questions table: stores custom RSVP questions per event
CREATE TABLE IF NOT EXISTS rsvp_questions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    question TEXT NOT NULL,
    type VARCHAR(20) NOT NULL, -- text, choice, multichoice, number
    options JSONB DEFAULT '[]',
    required BOOLEAN NOT NULL DEFAULT false,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_rsvp_questions_event ON rsvp_questions(event_id);
CREATE INDEX IF NOT EXISTS idx_rsvp_questions_sort ON rsvp_questions(event_id, sort_order);

-- rsvp_question_answers table: stores answers to custom RSVP questions
CREATE TABLE IF NOT EXISTS rsvp_question_answers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rsvp_id UUID NOT NULL REFERENCES rsvp_responses(id) ON DELETE CASCADE,
    question_id UUID NOT NULL REFERENCES rsvp_questions(id) ON DELETE CASCADE,
    answer TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_rsvp_answers_rsvp ON rsvp_question_answers(rsvp_id);
CREATE INDEX IF NOT EXISTS idx_rsvp_answers_question ON rsvp_question_answers(question_id);

-- Unique constraint: one answer per question per RSVP
CREATE UNIQUE INDEX IF NOT EXISTS idx_rsvp_answers_unique ON rsvp_question_answers(rsvp_id, question_id);

-- +goose StatementEnd
