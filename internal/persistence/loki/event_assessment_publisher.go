package loki

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"probability-engine/internal/domain"
)

type EventAssessmentPublisher struct {
	client PushClient
	job    string
	env    string
	now    func() time.Time
}

func NewEventAssessmentPublisher(
	client PushClient,
	job string,
	env string,
	now func() time.Time,
) *EventAssessmentPublisher {
	if job == "" {
		job = "probability-engine"
	}
	if env == "" {
		env = "dev"
	}
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}

	return &EventAssessmentPublisher{
		client: client,
		job:    job,
		env:    env,
		now:    now,
	}
}

func (p *EventAssessmentPublisher) PublishEventAssessments(ctx context.Context, assessments []domain.EventAssessment) error {
	if len(assessments) == 0 {
		return nil
	}

	streamsByState := make(map[string][]PushValue)

	for _, a := range assessments {
		line, err := marshalAssessmentLine(a)
		if err != nil {
			return fmt.Errorf("marshal event assessment %s: %w", a.ID, err)
		}

		state := string(a.Decision.State)
		if state == "" {
			state = "unknown"
		}

		ts := a.AssessedAt
		if ts.IsZero() {
			ts = p.now()
		}

		streamsByState[state] = append(streamsByState[state], PushValue{
			Timestamp: ts,
			Line:      line,
		})
	}

	streams := make([]PushStream, 0, len(streamsByState))
	for state, values := range streamsByState {
		streams = append(streams, PushStream{
			Stream: map[string]string{
				"job":            p.job,
				"component":      "event_assessment",
				"env":            p.env,
				"decision_state": state,
			},
			Values: values,
		})
	}

	return p.client.Push(ctx, PushRequest{Streams: streams})
}

type assessmentLine struct {
	ID           string                        `json:"id"`
	RunID        string                        `json:"runId"`
	AssessedAt   time.Time                     `json:"assessedAt"`
	EventID      string                        `json:"eventId"`
	Title        string                        `json:"title"`
	Source       string                        `json:"source"`
	URL          string                        `json:"url,omitempty"`
	Published    time.Time                     `json:"published,omitempty"`
	Decision     domain.ClassificationDecision `json:"decision"`
	SignalsCount int                           `json:"signalsCount"`
	Classifiers  []classifierDecisionSummary   `json:"classifiers,omitempty"`
}

type classifierDecisionSummary struct {
	Classifier string               `json:"classifier"`
	State      domain.DecisionState `json:"state"`
	Accepted   bool                 `json:"accepted"`
}

func marshalAssessmentLine(a domain.EventAssessment) (string, error) {
	classifiers := make([]classifierDecisionSummary, 0, len(a.Classifiers))
	for _, c := range a.Classifiers {
		classifiers = append(classifiers, classifierDecisionSummary{
			Classifier: c.Classifier,
			State:      c.Decision.State,
			Accepted:   c.Decision.Accepted,
		})
	}

	payload := assessmentLine{
		ID:           a.ID,
		RunID:        a.RunID,
		AssessedAt:   a.AssessedAt,
		EventID:      a.Event.ID,
		Title:        a.Event.Title,
		Source:       a.Event.Source,
		URL:          a.Event.URL,
		Published:    a.Event.Published,
		Decision:     a.Decision,
		SignalsCount: len(a.Signals),
		Classifiers:  classifiers,
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
