package models

// User represents an authenticated user.
// UserType is "agent" or "contact" and is stored in session to distinguish portal vs agent auth.
type User struct {
	ID        int    `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email,omitempty"`
	UserType  string `json:"user_type,omitempty"` // "agent" | "contact"
}
