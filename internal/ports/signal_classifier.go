package ports

import (
	"context"

	"probability-engine/internal/domain"
)

type SignalClassifier interface {
	Name() string
	Classify(ctx context.Context, ev domain.Event, rules []domain.SignalRule) ([]domain.Signal, error)
}

type AssessingSignalClassifier interface {
	SignalClassifier
	Assess(ctx context.Context, ev domain.Event, rules []domain.SignalRule) (domain.ClassifierAssessmentResult, error)
}
