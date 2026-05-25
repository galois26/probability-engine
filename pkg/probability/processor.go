package probability

import "context"

// Processor is a higher-level public API for callers that want event-in,
// event-out processing rather than raw assessments.
type Processor interface {
	Name() string
	Process(ctx context.Context, events []Event) ([]Event, error)
}

type Mode string

const (
	ModeEnrichExisting Mode = "enrich_existing"
	ModeEmitAssessment Mode = "emit_assessment"
)

type ProcessorConfig struct {
	Mode          Mode
	EngineVersion string
	RuleVersion   string
}

type DefaultProcessor struct {
	engine Engine
	cfg    ProcessorConfig
}

func NewProcessor(engine Engine, cfg ProcessorConfig) *DefaultProcessor {
	if cfg.Mode == "" {
		cfg.Mode = ModeEnrichExisting
	}

	return &DefaultProcessor{
		engine: engine,
		cfg:    cfg,
	}
}

func (p *DefaultProcessor) Name() string {
	return "probability"
}

func (p *DefaultProcessor) Process(ctx context.Context, events []Event) ([]Event, error) {
	assessments, err := p.engine.Assess(ctx, events)
	if err != nil {
		return nil, err
	}

	switch p.cfg.Mode {
	case ModeEmitAssessment:
		return emitAssessmentEvents(events, assessments), nil
	default:
		return enrichEvents(events, assessments), nil
	}
}

func emitAssessmentEvents(events []Event, assessments []Assessment) []Event {
	var output []Event

	for _, assessment := range assessments {
		// Find the corresponding event for this assessment
		var event *Event
		for _, e := range events {
			if e.ID == assessment.Event.ID {
				event = &e
				break
			}
		}

		if event == nil {
			continue // No matching event found, skip this assessment
		}

		// Create a new event for the assessment
		assessmentEvent := Event{
			ID:          assessment.ID,
			Fingerprint: assessment.Event.Fingerprint,
			Source:      "probability-engine",
			Title:       "Assessment for " + event.Title,
			Summary:     "Assessment decision: " + assessment.Decision.State,
			URL:         "", // Could link to a dashboard or report
			PublishedAt: assessment.AssessedAt,
			Lang:        assessment.Event.Lang,
			Country:     assessment.Event.Country,
			Labels:      map[string]string{"event_id": assessment.Event.ID},
			Raw:         map[string]any{"assessment": assessment},
		}

		output = append(output, assessmentEvent)
	}

	return output
}

func enrichEvents(events []Event, assessments []Assessment) []Event {

	var output []Event
	for _, event := range events {
		enrichedEvent := event // Start with the original event

		// Find the corresponding assessment for this event
		for _, assessment := range assessments {
			if assessment.Event.ID == event.ID {
				// Enrich the event with assessment data
				if enrichedEvent.Raw == nil {
					enrichedEvent.Raw = make(map[string]any)
				}
				enrichedEvent.Raw["assessment"] = assessment
				break
			}
		}

		output = append(output, enrichedEvent)
	}

	return output
}
