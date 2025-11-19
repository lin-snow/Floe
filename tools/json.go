package tools

import (
	"context"
	"encoding/json"
	"fmt"
)

type ParseJSONTool struct{}

func (t *ParseJSONTool) Run(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	sourceVal, ok := input["source"]
	if !ok {
		return nil, fmt.Errorf("missing required input 'source'")
	}
	source, ok := sourceVal.(string)
	if !ok {
		return nil, fmt.Errorf("'source' must be a string")
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(source), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return result, nil
}

func init() {
	Register("parse_json", &ParseJSONTool{})
}
