package domain

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// JSONMap is a map type that can be stored as JSONB in PostgreSQL.
type JSONMap map[string]interface{}

// Scan implements sql.Scanner interface for JSONMap.
func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	switch v := value.(type) {
	case []byte:
		if len(v) == 0 {
			*j = nil
			return nil
		}
		return json.Unmarshal(v, j)
	case string:
		if v == "" {
			*j = nil
			return nil
		}
		return json.Unmarshal([]byte(v), j)
	default:
		return errors.New("invalid type for JSONMap")
	}
}

// Value implements driver.Valuer interface for JSONMap.
func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// StringPtr returns a pointer to the provided string.
func StringPtr(s string) *string {
	return &s
}

// TimePtr returns a pointer to the provided time.Time.
func TimePtr(t time.Time) *time.Time {
	return &t
}
