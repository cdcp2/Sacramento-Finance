package idgen

import "github.com/google/uuid"

// New generates a new random UUID (v4).
// Switch to v7 when google/uuid adds stable v7 support.
func New() uuid.UUID {
	return uuid.New()
}

// NewString returns the UUID as a string.
func NewString() string {
	return uuid.New().String()
}

// Parse parses a UUID string, returns error if invalid.
func Parse(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}
