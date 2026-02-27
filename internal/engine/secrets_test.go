package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSecrets(t *testing.T) {
	dir := t.TempDir()
	content := `# Database config
DB_HOST=localhost
DB_PORT=5432
DB_PASSWORD="super secret"
API_KEY='sk-test-123'

# Empty line above is fine
SIMPLE=value
`
	path := filepath.Join(dir, ".env")
	os.WriteFile(path, []byte(content), 0644)

	secrets, err := LoadSecrets(path)
	if err != nil {
		t.Fatalf("LoadSecrets error: %v", err)
	}

	tests := map[string]string{
		"DB_HOST":     "localhost",
		"DB_PORT":     "5432",
		"DB_PASSWORD": "super secret",
		"API_KEY":     "sk-test-123",
		"SIMPLE":      "value",
	}

	for key, expected := range tests {
		if got := secrets[key]; got != expected {
			t.Errorf("secrets[%q] = %q, want %q", key, got, expected)
		}
	}

	if len(secrets) != 5 {
		t.Errorf("expected 5 secrets, got %d", len(secrets))
	}
}

func TestLoadSecretsInvalidFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	os.WriteFile(path, []byte("INVALID LINE WITHOUT EQUALS\n"), 0644)

	_, err := LoadSecrets(path)
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
}
