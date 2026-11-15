package links

import (
	"errors"
	"time"
)

var (
	ErrNotFound    = errors.New("link not found")
	ErrCodeExists  = errors.New("link code already exists")
)

type Link struct {
	Code      string    `json:"code"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
}
