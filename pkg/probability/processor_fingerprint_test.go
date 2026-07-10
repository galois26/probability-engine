package probability

import (
	"context"
	"testing"
	"time"
)

type stubEngine struct {
	assessments []Assessment
	err         error
}

func (s stubEngine) Assess(ctx context.Context, events []Event) ([]Assessment, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.assessments, nil
}

func TestProcessorEnrichExistingPreservesOriginalEventAndAddsAssessment(t *testing.T) {
	event := Event{
		ID:          "event-1",
		Fingerprint: "event-fingerprint",
		Source:      "newswire",
		Title:       "Port strike disrupts shipping",
	}
	assessment := Assessment{
		ID:         "assessment-1",
		EventID:    event.ID,
		AssessedAt: time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC),
		Event: EventSnapshot{
			ID:          event.ID,
			Fingerprint: event.Fingerprint,
			Title:       event.Title,
		},
		Decision: ClassificationDecision{State: "accepted_signal", Accepted: true},
	}

	processor := NewProcessor(stubEngine{assessments: []Assessment{assessment}}, ProcessorConfig{})

	out, err := processor.Process(context.Background(), []Event{event})
	if err != nil {
		t.Fatalf("Process returned error: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected one output event, got %d", len(out))
	}
	if got := out[0].Fingerprint; got != event.Fingerprint {
		t.Fatalf("output event fingerprint = %q, want original fingerprint %q", got, event.Fingerprint)
	}

	rawAssessment, ok := out[0].Raw["assessment"]
	if !ok {
		t.Fatal("expected enriched event Raw to contain assessment")
	}
	enrichedAssessment, ok := rawAssessment.(Assessment)
	if !ok {
		t.Fatalf("Raw assessment has type %T, want probability.Assessment", rawAssessment)
	}
	if got := enrichedAssessment.Event.Fingerprint; got != event.Fingerprint {
		t.Fatalf("enriched assessment fingerprint = %q, want %q", got, event.Fingerprint)
	}
}

func TestProcessorEmitAssessmentCopiesAssessmentFingerprint(t *testing.T) {
	event := Event{
		ID:          "event-1",
		Fingerprint: "event-fingerprint",
		Source:      "newswire",
		Title:       "Port strike disrupts shipping",
		Lang:        "en",
		Country:     "GB",
	}
	assessment := Assessment{
		ID:         "assessment-1",
		EventID:    event.ID,
		AssessedAt: time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC),
		Event: EventSnapshot{
			ID:          event.ID,
			Fingerprint: event.Fingerprint,
			Title:       event.Title,
			Lang:        event.Lang,
			Country:     event.Country,
		},
		Decision: ClassificationDecision{State: "accepted_signal", Accepted: true},
	}

	processor := NewProcessor(
		stubEngine{assessments: []Assessment{assessment}},
		ProcessorConfig{Mode: ModeEmitAssessment},
	)

	out, err := processor.Process(context.Background(), []Event{event})
	if err != nil {
		t.Fatalf("Process returned error: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected one emitted assessment event, got %d", len(out))
	}
	if got := out[0].ID; got != assessment.ID {
		t.Fatalf("emitted event ID = %q, want assessment ID %q", got, assessment.ID)
	}
	if got := out[0].Fingerprint; got != event.Fingerprint {
		t.Fatalf("emitted event fingerprint = %q, want %q", got, event.Fingerprint)
	}
	if got := out[0].Source; got != "probability-engine" {
		t.Fatalf("emitted event source = %q, want probability-engine", got)
	}
}
