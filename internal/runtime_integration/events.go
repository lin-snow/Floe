package runtime_integration

import (
	"time"
)

type EventType string

const (
	EventWorkflowStarted EventType = "workflow_started"
	EventSuperstepStart  EventType = "superstep_start"
	EventStepStart       EventType = "step_start"
	EventStepEnd         EventType = "step_end"
	EventStepSkipped     EventType = "step_skipped"
	EventMemoryUpdate    EventType = "memory_update"
	EventWorkflowEnd     EventType = "workflow_end"
	EventTraceSnapshot   EventType = "trace_snapshot"
	EventLog             EventType = "log"
)

type Event struct {
	Type      EventType              `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload"`
}

// NewEvent creates a new event with the current timestamp
func NewEvent(eventType EventType, payload map[string]interface{}) Event {
	return Event{
		Type:      eventType,
		Timestamp: time.Now(),
		Payload:   payload,
	}
}
