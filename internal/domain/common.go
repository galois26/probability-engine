package domain

import "time"

type Direction string

const (
	DirectionPositive Direction = "positive"
	DirectionNegative Direction = "negative"
	DirectionNeutral  Direction = "neutral"
	DirectionMixed    Direction = "mixed"
	DirectionUnknown  Direction = "unknown"
)

type InsightStatus string

const (
	InsightStatusOpen          InsightStatus = "open"
	InsightStatusAcknowledged  InsightStatus = "acknowledged"
	InsightStatusSilenced      InsightStatus = "silenced"
	InsightStatusDuplicate     InsightStatus = "duplicate"
	InsightStatusFalsePositive InsightStatus = "false_positive"
	InsightStatusClosed        InsightStatus = "closed"
)

type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

type TimeWindow struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}
