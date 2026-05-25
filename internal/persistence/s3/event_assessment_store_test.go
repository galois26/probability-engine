package s3

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/galois/probability-engine/internal/domain"
)

func TestEventAssessmentStore_SaveEventAssessments_WritesLatestAndRunSnapshot(t *testing.T) {
	now := time.Date(2026, 4, 11, 10, 0, 0, 0, time.UTC)
	fake := newFakeS3Client()

	store := &Store{
		client: fake,
		bucket: "test-bucket",
		prefix: "probability-engine",
		env:    "dev",
		clock:  fixedClock{now: now},
	}

	assessments := []domain.EventAssessment{
		{
			ID:         "2026-04-11T10:00:00Z:ev1",
			RunID:      "2026-04-11T10:00:00Z",
			AssessedAt: now,
			Event: domain.EventSnapshot{
				ID:        "ev1",
				Source:    "newsdata",
				Title:     "Sanctions announced on exports",
				Summary:   "Government expands export restrictions.",
				URL:       "https://example.com/1",
				Published: now.Add(-1 * time.Hour),
				Lang:      "en",
				Country:   "AA",
				Labels:    map[string]string{"category": "geopolitics"},
			},
			Decision: domain.ClassificationDecision{
				State:        domain.DecisionAcceptedSignal,
				Accepted:     true,
				PrimaryClass: "sanctions",
				Confidence:   0.81,
				Threshold:    0.63,
				Reasons:      []string{"at least one classifier emitted a signal"},
			},
			Signals: []domain.SignalSnapshot{
				{
					ID:          "sig1",
					EventID:     "ev1",
					Kind:        "sanctions",
					MarketScope: []string{"fx", "commodities"},
					Direction:   domain.DirectionNegative,
					Probability: 0.81,
					Classifier:  "naive_bayes",
					CreatedAt:   now,
					Trace: domain.SignalTrace{
						EventID:      "ev1",
						ModelVersion: "test-v1",
					},
				},
			},
			Classifiers: []domain.ClassifierAssessmentResult{
				{
					Classifier: "naive_bayes",
					Features: domain.FeatureAssessment{
						HasFeatures: true,
						Tokens:      []string{"sanctions", "export", "restrictions"},
						Keywords:    []string{"sanctions", "export"},
						Reason:      "feature extraction succeeded",
					},
					Rules: domain.RuleAssessment{
						Evaluated: false,
						Matched:   false,
						Reason:    "not applicable for naive bayes classifier",
					},
					NaiveBayes: domain.NaiveBayesAssessment{
						Evaluated:      true,
						PredictedClass: "sanctions",
						Scores: []domain.ClassScore{
							{Class: "sanctions", Score: -1.2, Probability: 0.81},
							{Class: "supply_chain", Score: -3.1, Probability: 0.09},
						},
						Reason: "top class accepted after class gate and threshold",
					},
					Decision: domain.ClassificationDecision{
						State:        domain.DecisionAcceptedSignal,
						Accepted:     true,
						PrimaryClass: "sanctions",
						Confidence:   0.81,
						Threshold:    0.63,
						Reasons:      []string{"top class probability met threshold"},
					},
					Signals: []domain.Signal{
						{
							ID:          "sig1",
							EventID:     "ev1",
							Kind:        "sanctions",
							MarketScope: []string{"fx", "commodities"},
							Direction:   domain.DirectionNegative,
							Probability: 0.81,
							Classifier:  "naive_bayes",
							CreatedAt:   now,
							Trace: domain.SignalTrace{
								EventID:      "ev1",
								ModelVersion: "test-v1",
							},
						},
					},
				},
			},
		},
		{
			ID:         "2026-04-11T10:00:00Z:ev2",
			RunID:      "2026-04-11T10:00:00Z",
			AssessedAt: now,
			Event: domain.EventSnapshot{
				ID:        "ev2",
				Source:    "newsdata",
				Title:     "Local weather remains stable",
				Summary:   "No significant market signal detected.",
				URL:       "https://example.com/2",
				Published: now.Add(-30 * time.Minute),
			},
			Decision: domain.ClassificationDecision{
				State:    domain.DecisionRejectedNoMatch,
				Accepted: false,
				Reasons:  []string{"no classifier emitted a signal"},
			},
			Classifiers: []domain.ClassifierAssessmentResult{
				{
					Classifier: "rules",
					Features: domain.FeatureAssessment{
						HasFeatures: true,
						Tokens:      []string{"local", "weather", "stable"},
						Reason:      "extracted features from title/summary",
					},
					Rules: domain.RuleAssessment{
						Evaluated: true,
						Matched:   false,
						Reason:    "no rules matched above threshold",
					},
					NaiveBayes: domain.NaiveBayesAssessment{
						Evaluated: false,
						Reason:    "not applicable for rules classifier",
					},
					Decision: domain.ClassificationDecision{
						State:    domain.DecisionRejectedNoMatch,
						Accepted: false,
						Reasons:  []string{"no rules matched above threshold"},
					},
				},
			},
		},
	}

	if err := store.SaveEventAssessments(context.Background(), assessments); err != nil {
		t.Fatalf("SaveEventAssessments() error = %v", err)
	}

	latestKey := "probability-engine/dev/events/latest.json"
	rawLatest, ok := fake.objects[latestKey]
	if !ok {
		t.Fatalf("expected latest object %q to be written", latestKey)
	}

	runKey := store.runKey("events", now)
	rawRun, ok := fake.objects[runKey]
	if !ok {
		t.Fatalf("expected run object %q to be written", runKey)
	}

	var latestPayload EventAssessmentSnapshot
	if err := json.Unmarshal(rawLatest, &latestPayload); err != nil {
		t.Fatalf("unmarshal latest payload: %v", err)
	}

	var runPayload EventAssessmentSnapshot
	if err := json.Unmarshal(rawRun, &runPayload); err != nil {
		t.Fatalf("unmarshal run payload: %v", err)
	}

	if latestPayload.Version != "v1" {
		t.Fatalf("latest Version = %q, want v1", latestPayload.Version)
	}
	if !latestPayload.GeneratedAt.Equal(now) {
		t.Fatalf("latest GeneratedAt = %s, want %s", latestPayload.GeneratedAt, now)
	}
	if latestPayload.Count != 2 {
		t.Fatalf("latest Count = %d, want 2", latestPayload.Count)
	}
	if len(latestPayload.Items) != 2 {
		t.Fatalf("latest item count = %d, want 2", len(latestPayload.Items))
	}

	if runPayload.Version != "v1" {
		t.Fatalf("run Version = %q, want v1", runPayload.Version)
	}
	if !runPayload.GeneratedAt.Equal(now) {
		t.Fatalf("run GeneratedAt = %s, want %s", runPayload.GeneratedAt, now)
	}
	if runPayload.Count != 2 {
		t.Fatalf("run Count = %d, want 2", runPayload.Count)
	}
	if len(runPayload.Items) != 2 {
		t.Fatalf("run item count = %d, want 2", len(runPayload.Items))
	}

	gotAccepted := latestPayload.Items[0]
	if gotAccepted.Event.ID != "ev1" {
		t.Fatalf("first event id = %q, want ev1", gotAccepted.Event.ID)
	}
	if gotAccepted.Decision.State != domain.DecisionAcceptedSignal {
		t.Fatalf("first decision state = %s, want %s", gotAccepted.Decision.State, domain.DecisionAcceptedSignal)
	}
	if len(gotAccepted.Signals) != 1 {
		t.Fatalf("first signals = %d, want 1", len(gotAccepted.Signals))
	}
	if len(gotAccepted.Classifiers) != 1 {
		t.Fatalf("first classifiers = %d, want 1", len(gotAccepted.Classifiers))
	}

	gotRejected := latestPayload.Items[1]
	if gotRejected.Event.ID != "ev2" {
		t.Fatalf("second event id = %q, want ev2", gotRejected.Event.ID)
	}
	if gotRejected.Decision.State != domain.DecisionRejectedNoMatch {
		t.Fatalf("second decision state = %s, want %s", gotRejected.Decision.State, domain.DecisionRejectedNoMatch)
	}
	if len(gotRejected.Signals) != 0 {
		t.Fatalf("second signals = %d, want 0", len(gotRejected.Signals))
	}
}

func TestEventAssessmentStore_SaveEventAssessments_PutError(t *testing.T) {
	fake := newFakeS3Client()
	fake.putErr = errors.New("put failed")

	store := &Store{
		client: fake,
		bucket: "test-bucket",
		prefix: "probability-engine",
		env:    "dev",
		clock:  fixedClock{now: time.Date(2026, 4, 11, 10, 0, 0, 0, time.UTC)},
	}

	err := store.SaveEventAssessments(context.Background(), []domain.EventAssessment{
		{
			ID:         "run:ev1",
			RunID:      "run",
			AssessedAt: time.Date(2026, 4, 11, 10, 0, 0, 0, time.UTC),
			Event: domain.EventSnapshot{
				ID:     "ev1",
				Source: "newsdata",
				Title:  "Test event",
			},
			Decision: domain.ClassificationDecision{
				State:    domain.DecisionRejectedNoMatch,
				Accepted: false,
			},
		},
	})
	if err == nil {
		t.Fatal("SaveEventAssessments() error = nil, want non-nil")
	}
}
