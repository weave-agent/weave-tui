package model

import "time"

// sessionHeader is a test-only copy of the JSONL session header shape.
type sessionHeader struct {
	Type      string    `json:"type"`
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	CWD       string    `json:"cwd"`
}
