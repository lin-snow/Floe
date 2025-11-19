package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"floe/internal/runtime_integration"
	"floe/runtime"
)

type Model struct {
	runtime *runtime.WorkflowRuntime
	sub     <-chan runtime_integration.Event

	// State
	steps      []StepItem
	activeStep string
	logs       []string
	variables  map[string]interface{}
	status     string

	// UI State
	width       int
	height      int
	selectedIdx int
}

type StepItem struct {
	ID     string
	Status string // pending, running, executed, skipped, failed
	Tool   string
}

func NewModel(rt *runtime.WorkflowRuntime) Model {
	// Initialize steps from workflow definition
	var steps []StepItem
	for _, s := range rt.Workflow().Steps {
		steps = append(steps, StepItem{
			ID:     s.ID,
			Status: "pending",
			Tool:   s.Tool,
		})
	}

	return Model{
		runtime:   rt,
		sub:       rt.Subscribe(),
		steps:     steps,
		variables: make(map[string]interface{}),
		status:    "Ready",
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		waitForEvent(m.sub),
		tick(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.selectedIdx > 0 {
				m.selectedIdx--
			}
		case "down", "j":
			if m.selectedIdx < len(m.steps)-1 {
				m.selectedIdx++
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case TickMsg:
		return m, tick()

	case EventMsg:
		m.handleEvent(msg)
		return m, waitForEvent(m.sub)
	}

	return m, nil
}

func (m *Model) handleEvent(e EventMsg) {
	switch e.Type {
	case runtime_integration.EventWorkflowStarted:
		m.status = "Running"
	case runtime_integration.EventWorkflowEnd:
		m.status = "Completed"
	case runtime_integration.EventStepStart:
		id := e.Payload["step_id"].(string)
		m.updateStepStatus(id, "running")
		m.activeStep = id
	case runtime_integration.EventStepEnd:
		id := e.Payload["step_id"].(string)
		status := e.Payload["status"].(string)
		m.updateStepStatus(id, status)
		if e.Payload["error"] != "" {
			m.logs = append(m.logs, fmt.Sprintf("[ERROR] Step %s: %s", id, e.Payload["error"]))
		}
	case runtime_integration.EventStepSkipped:
		id := e.Payload["step_id"].(string)
		m.updateStepStatus(id, "skipped")
	case runtime_integration.EventMemoryUpdate:
		key := e.Payload["key"].(string)
		val := e.Payload["value"]
		m.variables[key] = val
	}
}

func (m *Model) updateStepStatus(id, status string) {
	for i, s := range m.steps {
		if s.ID == id {
			m.steps[i].Status = status
			break
		}
	}
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Layout
	cols := []string{
		m.renderStepList(),
		m.renderDetails(),
		m.renderVariables(),
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, cols...)
}
