package loader

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFlow(t *testing.T) {
	dir := t.TempDir()
	content := `
name: test-flow
version: "1.0"
description: "A test flow"
input:
  properties:
    url:
      type: string
      required: true
trigger:
  type: webhook
  path: /test
steps:
  - name: step1
    connector: http
    action: request
    input:
      url: "${{ input.url }}"
    on_error: abort
`
	path := filepath.Join(dir, "test.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	flow, err := LoadFlow(path)
	if err != nil {
		t.Fatalf("LoadFlow error: %v", err)
	}

	if flow.Name != "test-flow" {
		t.Errorf("name = %q, want test-flow", flow.Name)
	}
	if flow.Version != "1.0" {
		t.Errorf("version = %q, want 1.0", flow.Version)
	}
	if len(flow.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(flow.Steps))
	}
	if flow.Steps[0].Connector != "http" {
		t.Errorf("step connector = %q, want http", flow.Steps[0].Connector)
	}
	if flow.Trigger == nil || flow.Trigger.Path != "/test" {
		t.Error("trigger not parsed correctly")
	}
}

func TestLoadFlowMissingName(t *testing.T) {
	dir := t.TempDir()
	content := `
steps:
  - name: s
    connector: log
    action: print
`
	path := filepath.Join(dir, "bad.yaml")
	os.WriteFile(path, []byte(content), 0644)

	_, err := LoadFlow(path)
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestLoadFlows(t *testing.T) {
	dir := t.TempDir()

	flow1 := `
name: flow-a
version: "1.0"
steps:
  - name: s
    connector: log
    action: print
    input:
      message: "a"
`
	flow2 := `
name: flow-b
version: "2.0"
steps:
  - name: s
    connector: log
    action: print
    input:
      message: "b"
`
	os.WriteFile(filepath.Join(dir, "a.yaml"), []byte(flow1), 0644)
	os.WriteFile(filepath.Join(dir, "b.yml"), []byte(flow2), 0644)
	os.WriteFile(filepath.Join(dir, "ignore.txt"), []byte("not a flow"), 0644)

	flows, err := LoadFlows(dir)
	if err != nil {
		t.Fatalf("LoadFlows error: %v", err)
	}

	if len(flows) != 2 {
		t.Fatalf("expected 2 flows, got %d", len(flows))
	}
	if flows["flow-a"] == nil {
		t.Error("flow-a not loaded")
	}
	if flows["flow-b"] == nil {
		t.Error("flow-b not loaded")
	}
}

func TestLoadFlowsDuplicateName(t *testing.T) {
	dir := t.TempDir()
	content := `
name: duplicate
steps:
  - name: s
    connector: log
    action: print
    input:
      message: "test"
`
	os.WriteFile(filepath.Join(dir, "a.yaml"), []byte(content), 0644)
	os.WriteFile(filepath.Join(dir, "b.yaml"), []byte(content), 0644)

	_, err := LoadFlows(dir)
	if err == nil {
		t.Fatal("expected error for duplicate flow names")
	}
}
