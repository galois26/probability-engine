package domain

import "time"

type Event struct {
	ID          string                 `json:"id"`
	Fingerprint string                 `json:"fingerprint"`
	Source      string                 `json:"source"`
	Title       string                 `json:"title"`
	Summary     string                 `json:"summary"`
	URL         string                 `json:"url"`
	Published   time.Time              `json:"published_at"`
	Lang        string                 `json:"lang"`
	Country     string                 `json:"country"`
	Labels      map[string]string      `json:"labels,omitempty"`
	Raw         map[string]interface{} `json:"raw,omitempty"`
}
