package probability

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/galois26/probability-engine/internal/domain"
)

type recordingAssessor struct {
	events []domain.Event
	err    error
	calls  int
}

func (r *recordingAssessor) AssessEvents(ctx context.Context, events []domain.Event) ([]domain.EventAssessment, error) {
	r.calls++
	r.events = append([]domain.Event(nil), events...)
	if r.err != nil {
		return nil, r.err
	}

	assessments := make([]domain.EventAssessment, 0, len(events))
	for _, ev := range events {
		assessments = append(assessments, domain.EventAssessment{
			ID:         "assessment-" + ev.ID,
			AssessedAt: time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC),
			Event: domain.EventSnapshot{
				ID:          ev.ID,
				Fingerprint: ev.Fingerprint,
				Source:      ev.Source,
				Title:       ev.Title,
				Summary:     ev.Summary,
				URL:         ev.URL,
				Published:   ev.Published,
				Lang:        ev.Lang,
				Country:     ev.Country,
				Labels:      ev.Labels,
				Raw:         ev.Raw,
			},
		})
	}

	return assessments, nil
}

func TestEngineAdapterAssessGeneratesAndPropagatesFingerprint(t *testing.T) {
	published := time.Date(2026, 7, 10, 9, 30, 0, 0, time.UTC)
	event := Event{
		ID:          "event-1",
		Source:      "newswire",
		Title:       "Port strike disrupts shipping",
		Summary:     "Major freight routes are delayed.",
		URL:         "https://example.com/story?utm_source=newsletter&id=123",
		PublishedAt: published,
		Lang:        "en",
		Country:     "GB",
		Labels:      map[string]string{"source_type": "news"},
		Raw:         map[string]any{"priority": "high"},
	}
	expectedFingerprint := EventFingerprint(event)

	assessor := &recordingAssessor{}
	adapter := NewEngineAdapter(assessor, "engine-test", "rules-test")

	assessments, err := adapter.Assess(context.Background(), []Event{event})
	if err != nil {
		t.Fatalf("Assess returned error: %v", err)
	}

	if assessor.calls != 1 {
		t.Fatalf("expected assessor to be called once, got %d", assessor.calls)
	}
	if len(assessor.events) != 1 {
		t.Fatalf("expected one domain event, got %d", len(assessor.events))
	}
	if got := assessor.events[0].Fingerprint; got != expectedFingerprint {
		t.Fatalf("domain event fingerprint = %q, want %q", got, expectedFingerprint)
	}
	if got := assessor.events[0].Published; !got.Equal(published) {
		t.Fatalf("domain event published = %v, want %v", got, published)
	}

	if len(assessments) != 1 {
		t.Fatalf("expected one assessment, got %d", len(assessments))
	}
	if got := assessments[0].Event.Fingerprint; got != expectedFingerprint {
		t.Fatalf("assessment event fingerprint = %q, want %q", got, expectedFingerprint)
	}
	if got := assessments[0].Event.Published; !got.Equal(published) {
		t.Fatalf("assessment event published = %v, want %v", got, published)
	}
	if got := assessments[0].EngineVersion; got != "engine-test" {
		t.Fatalf("engine version = %q, want %q", got, "engine-test")
	}
	if got := assessments[0].RuleVersion; got != "rules-test" {
		t.Fatalf("rule version = %q, want %q", got, "rules-test")
	}
}

func TestEngineAdapterAssessPreservesCallerProvidedFingerprint(t *testing.T) {
	event := Event{
		ID:          "event-2",
		Fingerprint: "caller-fingerprint",
		Source:      "newswire",
		Title:       "Export restrictions announced",
		Summary:     "New export controls affect semiconductor supply.",
	}

	assessor := &recordingAssessor{}
	adapter := NewEngineAdapter(assessor, "engine-test", "rules-test")

	assessments, err := adapter.Assess(context.Background(), []Event{event})
	if err != nil {
		t.Fatalf("Assess returned error: %v", err)
	}

	if got := assessor.events[0].Fingerprint; got != "caller-fingerprint" {
		t.Fatalf("domain event fingerprint = %q, want caller-provided fingerprint", got)
	}
	if got := assessments[0].Event.Fingerprint; got != "caller-fingerprint" {
		t.Fatalf("assessment event fingerprint = %q, want caller-provided fingerprint", got)
	}
}

func TestEngineAdapterAssessEmptyInputDoesNotCallAssessor(t *testing.T) {
	assessor := &recordingAssessor{}
	adapter := NewEngineAdapter(assessor, "engine-test", "rules-test")

	assessments, err := adapter.Assess(context.Background(), nil)
	if err != nil {
		t.Fatalf("Assess returned error: %v", err)
	}
	if len(assessments) != 0 {
		t.Fatalf("expected no assessments, got %d", len(assessments))
	}
	if assessor.calls != 0 {
		t.Fatalf("expected assessor not to be called, got %d calls", assessor.calls)
	}
}

func TestEngineAdapterAssessReturnsNilEngineError(t *testing.T) {
	adapter := NewEngineAdapter(nil, "engine-test", "rules-test")

	_, err := adapter.Assess(context.Background(), []Event{{ID: "event-1"}})
	if err == nil {
		t.Fatal("expected error for nil internal engine")
	}
}

func TestEngineAdapterAssessReturnsAssessorError(t *testing.T) {
	assessorErr := errors.New("assessor failed")
	adapter := NewEngineAdapter(&recordingAssessor{err: assessorErr}, "engine-test", "rules-test")

	_, err := adapter.Assess(context.Background(), []Event{{ID: "event-1"}})
	if !errors.Is(err, assessorErr) {
		t.Fatalf("error = %v, want %v", err, assessorErr)
	}
}

func TestToDomainEventPreservesZeroPublishedAt(t *testing.T) {
	event := Event{
		ID:          "event-zero-time",
		Fingerprint: "fp-zero-time",
		Title:       "No timestamp yet",
	}

	domainEvent := toDomainEvent(event)
	if !domainEvent.Published.IsZero() {
		t.Fatalf("Published = %v, want zero time", domainEvent.Published)
	}
	if got := domainEvent.Fingerprint; got != "fp-zero-time" {
		t.Fatalf("Fingerprint = %q, want %q", got, "fp-zero-time")
	}
}

func TestFromDomainEventSnapshotCopiesFingerprint(t *testing.T) {
	snapshot := domain.EventSnapshot{
		ID:          "event-1",
		Fingerprint: "snapshot-fingerprint",
		Source:      "newswire",
		Title:       "Snapshot title",
	}

	publicSnapshot := fromDomainEventSnapshot(snapshot)
	if got := publicSnapshot.Fingerprint; got != "snapshot-fingerprint" {
		t.Fatalf("Fingerprint = %q, want snapshot fingerprint", got)
	}
}
