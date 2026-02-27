package engine

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// LoadSecrets reads a .env-style secrets file (KEY=VALUE per line).
// Lines starting with # are comments. Empty lines are skipped.
func LoadSecrets(path string) (map[string]string, error) {
	secrets := make(map[string]string)

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening secrets file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("secrets file line %d: invalid format (expected KEY=VALUE)", lineNum)
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		// Strip surrounding quotes.
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		secrets[key] = value
	}

	return secrets, scanner.Err()
}
