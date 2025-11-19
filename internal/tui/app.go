package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"floe/internal/runtime_integration"
	"floe/runtime"
)

type App struct {
	runtime *runtime.WorkflowRuntime
	program *tea.Program
}

func NewApp(rt *runtime.WorkflowRuntime) *App {
	return &App{
		runtime: rt,
	}
}

func (a *App) Run() error {
	// Start runtime in a separate goroutine
	go func() {
		if err := a.runtime.Run(); err != nil {
			// Handle runtime error (maybe send an event or log)
			fmt.Printf("Runtime error: %v\n", err)
		}
	}()

	// Initialize Bubbletea model
	initialModel := NewModel(a.runtime)
	a.program = tea.NewProgram(initialModel, tea.WithAltScreen())

	if _, err := a.program.Run(); err != nil {
		return err
	}
	return nil
}

// TickMsg is sent to update the UI periodically
type TickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(time.Millisecond*50, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// EventMsg wraps runtime events for Bubbletea
type EventMsg runtime_integration.Event

func waitForEvent(sub <-chan runtime_integration.Event) tea.Cmd {
	return func() tea.Msg {
		return EventMsg(<-sub)
	}
}
