package models

import "time"

// DefaultStatusIDs are the fixed IDs of the four default statuses created in schema (Open, Snoozed, Resolved, Closed).
// They are used to prevent deletion and to identify default semantics in conversation SQL; names remain editable.
var DefaultStatusIDs = []int{1, 2, 3, 4}

// Default status ID constants for use in conversation/SLA logic (schema insert order).
const (
	DefaultStatusIDOpen     = 1
	DefaultStatusIDSnoozed  = 2
	DefaultStatusIDResolved = 3
	DefaultStatusIDClosed   = 4
)

type Status struct {
	ID        int       `db:"id" json:"id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	Name      string    `db:"name" json:"name"`
	IsDefault bool      `json:"is_default"` // set by backend from DefaultStatusIDs; not stored in DB
}
