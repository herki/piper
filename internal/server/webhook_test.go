package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"piper/internal/engine"
	"piper/internal/plugin"
	"piper/internal/plugin/builtin"
	"piper/internal/types"
)

func testSetup() *WebhookServer {
	registry := plugin.NewRegistry()
	registry.Register(builtin.NewLogConnector())
	registry.Register(builtin.NewShellConnector())

	eng := engine.NewEngine(registry)

	flows := map[string]*types.FlowDef{
		"test-flow": {
			Name:    "test-flow",
			Trigger: &types.TriggerDef{Type: "webhook", Path: "/test"},
			Steps: []types.StepDef{
				{
					Name:      "greet",
					Connector: "log",
					Action:    "print",
					Input:     map[string]any{"message": "Hello ${{ input.name }}"},
				},
			},
		},
	}

	return NewWebhookServer(eng, flows)
}

func TestHealthEndpoint(t *testing.T) {
	srv := testSetup()
	mux := http.NewServeMux()
	mux.HandleFunc("/health", srv.handleHealth)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Errorf("body status = %q, want ok", body["status"])
	}
}

func TestTriggerFlow(t *testing.T) {
	srv := testSetup()
	mux := http.NewServeMux()
	mux.HandleFunc("/", srv.handleTrigger)

	input := map[string]any{"name": "World"}
	body, _ := json.Marshal(input)

	req := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var result types.FlowResult
	json.NewDecoder(w.Body).Decode(&result)
	if result.Status != "success" {
		t.Errorf("flow status = %q, want success", result.Status)
	}
	if result.Flow != "test-flow" {
		t.Errorf("flow name = %q, want test-flow", result.Flow)
	}
}

func TestTriggerNotFound(t *testing.T) {
	srv := testSetup()
	mux := http.NewServeMux()
	mux.HandleFunc("/", srv.handleTrigger)

	req := httptest.NewRequest("POST", "/nonexistent", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestTriggerMethodNotAllowed(t *testing.T) {
	srv := testSetup()
	mux := http.NewServeMux()
	mux.HandleFunc("/", srv.handleTrigger)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 405 {
		t.Errorf("status = %d, want 405", w.Code)
	}
}
