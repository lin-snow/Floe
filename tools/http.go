package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

type HTTPGetTool struct{}

func (t *HTTPGetTool) Run(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	urlVal, ok := input["url"]
	if !ok {
		return nil, fmt.Errorf("missing required input 'url'")
	}
	url, ok := urlVal.(string)
	if !ok {
		return nil, fmt.Errorf("'url' must be a string")
	}

	// Create a client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return string(body), nil
}

func init() {
	Register("http_get", &HTTPGetTool{})
}
