package s3

import (
	"time"

	"github.com/galois26/probability-engine/internal/domain"
)

type SignalSnapshot struct {
	Version     string          `json:"version"`
	GeneratedAt time.Time       `json:"generatedAt"`
	Count       int             `json:"count"`
	Items       []domain.Signal `json:"items"`
}

type InsightSnapshot struct {
	Version     string           `json:"version"`
	GeneratedAt time.Time        `json:"generatedAt"`
	Count       int              `json:"count"`
	Items       []domain.Insight `json:"items"`
}

type EventAssessmentSnapshot struct {
	Version     string                   `json:"version"`
	GeneratedAt time.Time                `json:"generatedAt"`
	Count       int                      `json:"count"`
	Items       []domain.EventAssessment `json:"items"`
}
