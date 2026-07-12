-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_rsvp_answers_unique;
DROP INDEX IF EXISTS idx_rsvp_answers_question;
DROP INDEX IF EXISTS idx_rsvp_answers_rsvp;
DROP TABLE IF EXISTS rsvp_question_answers;

DROP INDEX IF EXISTS idx_rsvp_questions_sort;
DROP INDEX IF EXISTS idx_rsvp_questions_event;
DROP TABLE IF EXISTS rsvp_questions;

-- +goose StatementEnd
