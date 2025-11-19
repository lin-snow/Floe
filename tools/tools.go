package tools

import (
	"context"
	"fmt"
)

// Tool is the interface that all tools must implement.
type Tool interface {
	Run(ctx context.Context, input map[string]interface{}) (interface{}, error)
}

// Registry stores available tools.
var Registry = make(map[string]Tool)

// Register adds a tool to the registry.
func Register(name string, tool Tool) {
	Registry[name] = tool
}

// Get retrieves a tool by name.
func Get(name string) (Tool, error) {
	tool, ok := Registry[name]
	if !ok {
		return nil, fmt.Errorf("tool '%s' not found", name)
	}
	return tool, nil
}
