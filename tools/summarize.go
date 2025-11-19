package tools

import (
	"context"
	"fmt"
	"strings"
)

type SummarizeTool struct{}

func (t *SummarizeTool) Run(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	textVal, ok := input["text"]
	if !ok {
		return nil, fmt.Errorf("missing required input 'text'")
	}
	text, ok := textVal.(string)
	if !ok {
		return nil, fmt.Errorf("'text' must be a string")
	}

	// Mock summary: just count words and return a string
	words := strings.Fields(text)
	summary := fmt.Sprintf("Summary: Text contains %d words. Preview: %s...", len(words), truncate(text, 50))

	return summary, nil
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max]
	}
	return s
}

func init() {
	Register("summarize", &SummarizeTool{})
}
