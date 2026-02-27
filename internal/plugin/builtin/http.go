package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"piper/internal/plugin"
	"piper/internal/types"
)

// HTTPConnector makes generic HTTP requests.
type HTTPConnector struct{}

func NewHTTPConnector() *HTTPConnector { return &HTTPConnector{} }

func (h *HTTPConnector) Name() string { return "http" }

func (h *HTTPConnector) Actions() []plugin.ActionDef {
	return []plugin.ActionDef{
		{
			Name:        "request",
			Description: "Make an HTTP request",
			Input: map[string]types.FieldDef{
				"url":     {Type: "string", Description: "Request URL", Required: true},
				"method":  {Type: "string", Description: "HTTP method (GET, POST, PUT, DELETE)", Required: false},
				"headers": {Type: "object", Description: "Request headers", Required: false},
				"body":    {Type: "any", Description: "Request body (will be JSON-encoded if object)", Required: false},
			},
			Output: map[string]types.FieldDef{
				"status_code": {Type: "integer", Description: "HTTP status code"},
				"body":        {Type: "any", Description: "Response body"},
				"headers":     {Type: "object", Description: "Response headers"},
			},
		},
	}
}

func (h *HTTPConnector) Execute(ctx context.Context, action string, input map[string]any) (*types.StepResult, error) {
	if action != "request" {
		return nil, fmt.Errorf("http connector: unknown action %q", action)
	}

	url, _ := input["url"].(string)
	if url == "" {
		return nil, fmt.Errorf("http connector: 'url' is required")
	}

	method := "GET"
	if m, ok := input["method"].(string); ok && m != "" {
		method = strings.ToUpper(m)
	}

	var bodyReader io.Reader
	if body, ok := input["body"]; ok && body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("http connector: encoding body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("http connector: creating request: %w", err)
	}

	if headers, ok := input["headers"].(map[string]any); ok {
		for k, v := range headers {
			req.Header.Set(k, fmt.Sprintf("%v", v))
		}
	}

	if bodyReader != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http connector: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("http connector: reading response: %w", err)
	}

	var parsedBody any
	if err := json.Unmarshal(respBody, &parsedBody); err != nil {
		parsedBody = string(respBody)
	}

	respHeaders := make(map[string]any)
	for k, v := range resp.Header {
		if len(v) == 1 {
			respHeaders[k] = v[0]
		} else {
			respHeaders[k] = v
		}
	}

	output := map[string]any{
		"status_code": resp.StatusCode,
		"body":        parsedBody,
		"headers":     respHeaders,
	}

	status := "success"
	if resp.StatusCode >= 400 {
		status = "failed"
	}

	return &types.StepResult{
		Status: status,
		Output: output,
	}, nil
}

func (h *HTTPConnector) Validate() error { return nil }
