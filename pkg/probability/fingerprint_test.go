package probability

import (
	"strings"
	"testing"
	"time"
)

func TestEventFingerprintIsDeterministic(t *testing.T) {
	event := Event{
		Source:      "Reuters",
		Title:       "Shipping Disrupted",
		Summary:     "Freight delays reported",
		URL:         "https://example.com/story?id=123&utm_source=newsletter",
		PublishedAt: time.Date(2026, 7, 10, 14, 35, 0, 0, time.UTC),
	}

	first := EventFingerprint(event)
	second := EventFingerprint(event)

	if first == "" {
		t.Fatal("fingerprint must not be empty")
	}
	if first != second {
		t.Fatalf("fingerprint changed between calls: %q != %q", first, second)
	}
}

func TestEventFingerprintNormalizesTextAndTrackingParams(t *testing.T) {
	published := time.Date(2026, 7, 10, 14, 35, 0, 0, time.UTC)

	left := Event{
		Source:      " Reuters ",
		Title:       " Shipping   Disrupted ",
		Summary:     "Freight delays reported",
		URL:         "https://example.com/story?id=123&utm_source=newsletter&utm_campaign=daily#section",
		PublishedAt: published,
	}
	right := Event{
		Source:      "reuters",
		Title:       "shipping disrupted",
		Summary:     "Freight delays reported",
		URL:         "https://example.com/story?id=123",
		PublishedAt: published,
	}

	if got, want := EventFingerprint(left), EventFingerprint(right); got != want {
		t.Fatalf("normalized fingerprints differ: got %q, want %q", got, want)
	}
}

func TestEventFingerprintUsesPublishedDayOnly(t *testing.T) {
	base := Event{
		Source:  "Reuters",
		Title:   "Shipping Disrupted",
		Summary: "Freight delays reported",
		URL:     "https://example.com/story?id=123",
	}

	morning := base
	morning.PublishedAt = time.Date(2026, 7, 10, 8, 0, 0, 0, time.UTC)

	evening := base
	evening.PublishedAt = time.Date(2026, 7, 10, 23, 59, 0, 0, time.UTC)

	nextDay := base
	nextDay.PublishedAt = time.Date(2026, 7, 11, 0, 1, 0, 0, time.UTC)

	if EventFingerprint(morning) != EventFingerprint(evening) {
		t.Fatal("same UTC publish day should produce the same fingerprint")
	}
	if EventFingerprint(morning) == EventFingerprint(nextDay) {
		t.Fatal("different UTC publish days should produce different fingerprints")
	}
}

func TestAssessmentIDChangesWithEngineOrRuleVersion(t *testing.T) {
	fingerprint := "event-fingerprint"
	base := AssessmentID(fingerprint, "engine-v1", "rules-v1")

	if base == AssessmentID(fingerprint, "engine-v2", "rules-v1") {
		t.Fatal("assessment ID should change when engine version changes")
	}
	if base == AssessmentID(fingerprint, "engine-v1", "rules-v2") {
		t.Fatal("assessment ID should change when rule version changes")
	}
}

func TestFingerprintLengthIsSHA256Hex(t *testing.T) {
	fingerprint := EventFingerprint(Event{Title: "test"})
	if len(fingerprint) != 64 {
		t.Fatalf("fingerprint length = %d, want 64", len(fingerprint))
	}
	if strings.ToLower(fingerprint) != fingerprint {
		t.Fatalf("fingerprint should be lowercase hex: %q", fingerprint)
	}
}
