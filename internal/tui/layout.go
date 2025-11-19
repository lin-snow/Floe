package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Styles
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#25A065")).
			Padding(0, 1)

	stepStyle = lipgloss.NewStyle().
			PaddingLeft(1).
			PaddingRight(1).
			Foreground(lipgloss.Color("241"))

	selectedStepStyle = lipgloss.NewStyle().
				PaddingLeft(1).
				PaddingRight(1).
				Foreground(lipgloss.Color("205")).
				Bold(true)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	runningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214")) // Orange
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))  // Green
	failedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("196")) // Red
	skippedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("243")) // Grey

	borderStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63"))
)

func (m Model) renderStepList() string {
	var s strings.Builder
	s.WriteString(titleStyle.Render("Steps"))
	s.WriteString("\n\n")

	for i, step := range m.steps {
		cursor := " "
		style := stepStyle

		if m.selectedIdx == i {
			cursor = ">"
			style = selectedStepStyle
		}

		statusIcon := "○"
		statusColor := statusStyle

		switch step.Status {
		case "running":
			statusIcon = "●"
			statusColor = runningStyle
		case "executed":
			statusIcon = "✓"
			statusColor = successStyle
		case "skipped":
			statusIcon = "↷"
			statusColor = skippedStyle
		case "failed":
			statusIcon = "✗"
			statusColor = failedStyle
		}

		line := fmt.Sprintf("%s %s %s", cursor, statusColor.Render(statusIcon), style.Render(step.ID))
		s.WriteString(line + "\n")
	}

	return borderStyle.
		Width(30).
		Height(m.height - 2).
		Render(s.String())
}

func (m Model) renderDetails() string {
	var s strings.Builder
	s.WriteString(titleStyle.Render("Details"))
	s.WriteString("\n\n")

	if m.selectedIdx < len(m.steps) {
		step := m.steps[m.selectedIdx]
		s.WriteString(fmt.Sprintf("ID: %s\n", step.ID))
		s.WriteString(fmt.Sprintf("Tool: %s\n", step.Tool))
		s.WriteString(fmt.Sprintf("Status: %s\n", step.Status))
		s.WriteString("\n--- Logs ---\n")

		// Filter logs for this step (simple implementation)
		for _, log := range m.logs {
			if strings.Contains(log, step.ID) {
				s.WriteString(log + "\n")
			}
		}
	}

	return borderStyle.
		Width(m.width/2 - 15). // Approximate
		Height(m.height - 2).
		Render(s.String())
}

func (m Model) renderVariables() string {
	var s strings.Builder
	s.WriteString(titleStyle.Render("Variables"))
	s.WriteString("\n\n")

	// Sort keys for stable display
	keys := make([]string, 0, len(m.variables))
	for k := range m.variables {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		val := m.variables[k]
		valStr := fmt.Sprintf("%v", val)
		if len(valStr) > 50 {
			valStr = valStr[:47] + "..."
		}
		s.WriteString(fmt.Sprintf("%s: %s\n", k, valStr))
	}

	return borderStyle.
		Width(m.width/2 - 15). // Approximate
		Height(m.height - 2).
		Render(s.String())
}
