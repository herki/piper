package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"piper/internal/engine"
	"piper/internal/types"
)

// WebhookServer serves HTTP requests that trigger flows.
type WebhookServer struct {
	engine *engine.Engine
	flows  map[string]*types.FlowDef
	routes map[string]*types.FlowDef // trigger path -> flow
}

// NewWebhookServer creates a new webhook server.
func NewWebhookServer(eng *engine.Engine, flows map[string]*types.FlowDef) *WebhookServer {
	routes := make(map[string]*types.FlowDef)
	for _, f := range flows {
		if f.Trigger != nil && f.Trigger.Type == "webhook" {
			routes[f.Trigger.Path] = f
		}
	}
	return &WebhookServer{
		engine: eng,
		flows:  flows,
		routes: routes,
	}
}

// ListenAndServe starts the HTTP server.
func (s *WebhookServer) ListenAndServe(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/flows", s.handleListFlows)
	mux.HandleFunc("/", s.handleTrigger)
	return http.ListenAndServe(addr, mux)
}

func (s *WebhookServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *WebhookServer) handleListFlows(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	type flowInfo struct {
		Name        string      `json:"name"`
		Description string      `json:"description"`
		TriggerPath string      `json:"trigger_path,omitempty"`
		Input       interface{} `json:"input,omitempty"`
	}

	infos := make([]flowInfo, 0, len(s.flows))
	for _, f := range s.flows {
		fi := flowInfo{
			Name:        f.Name,
			Description: f.Description,
			Input:       f.Input,
		}
		if f.Trigger != nil {
			fi.TriggerPath = f.Trigger.Path
		}
		infos = append(infos, fi)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(infos)
}

func (s *WebhookServer) handleTrigger(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	flow, ok := s.routes[r.URL.Path]
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("no flow mapped to path %q", r.URL.Path),
		})
		return
	}

	var input map[string]any
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body: " + err.Error()})
			return
		}
	}
	if input == nil {
		input = make(map[string]any)
	}

	result, err := s.engine.Run(context.Background(), flow, input)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	statusCode := http.StatusOK
	if result.Status == "failed" {
		statusCode = http.StatusInternalServerError
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(result)
}
