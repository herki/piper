package engine

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode"

	"piper/internal/types"
)

var exprRegex = regexp.MustCompile(`\$\{\{\s*(.+?)\s*\}\}`)

// StepContext holds the state available during flow execution for variable resolution.
type StepContext struct {
	Input map[string]any
	Steps map[string]*types.StepResult
	Env   map[string]string
}

// NewStepContext creates a StepContext from flow input.
func NewStepContext(input map[string]any) *StepContext {
	env := make(map[string]string)
	for _, e := range os.Environ() {
		if k, v, ok := strings.Cut(e, "="); ok {
			env[k] = v
		}
	}
	return &StepContext{
		Input: input,
		Steps: make(map[string]*types.StepResult),
		Env:   env,
	}
}

// AddStepResult records the result of a step for later reference.
func (sc *StepContext) AddStepResult(name string, result *types.StepResult) {
	sc.Steps[name] = result
}

// ResolveMap recursively resolves all expressions in a map.
func (sc *StepContext) ResolveMap(m map[string]any) (map[string]any, error) {
	result := make(map[string]any, len(m))
	for k, v := range m {
		resolved, err := sc.resolveValue(v)
		if err != nil {
			return nil, fmt.Errorf("resolving %q: %w", k, err)
		}
		result[k] = resolved
	}
	return result, nil
}

func (sc *StepContext) resolveValue(v any) (any, error) {
	switch val := v.(type) {
	case string:
		return sc.resolveString(val)
	case map[string]any:
		return sc.ResolveMap(val)
	case []any:
		resolved := make([]any, len(val))
		for i, item := range val {
			r, err := sc.resolveValue(item)
			if err != nil {
				return nil, err
			}
			resolved[i] = r
		}
		return resolved, nil
	default:
		return v, nil
	}
}

// resolveString replaces all ${{ ... }} expressions in a string.
func (sc *StepContext) resolveString(s string) (any, error) {
	// If the entire string is a single expression, return the raw value (preserving type).
	if match := exprRegex.FindStringSubmatch(s); match != nil && match[0] == s {
		return sc.evaluateExpr(match[1])
	}

	// Otherwise, do string interpolation.
	var evalErr error
	result := exprRegex.ReplaceAllStringFunc(s, func(match string) string {
		sub := exprRegex.FindStringSubmatch(match)
		val, err := sc.evaluateExpr(sub[1])
		if err != nil {
			evalErr = err
			return match
		}
		return fmt.Sprintf("%v", val)
	})
	return result, evalErr
}

// evaluateExpr evaluates a single expression like "input.name | slugify".
func (sc *StepContext) evaluateExpr(expr string) (any, error) {
	parts := strings.SplitN(expr, "|", 2)
	path := strings.TrimSpace(parts[0])

	val, err := sc.resolvePath(path)
	if err != nil {
		return nil, err
	}

	if len(parts) == 2 {
		pipeFn := strings.TrimSpace(parts[1])
		val, err = applyPipe(val, pipeFn)
		if err != nil {
			return nil, err
		}
	}

	return val, nil
}

// resolvePath resolves a dotted path like "input.name" or "steps.create-repo.output.repo_url".
func (sc *StepContext) resolvePath(path string) (any, error) {
	segments := strings.SplitN(path, ".", 2)
	root := segments[0]

	switch root {
	case "input":
		if len(segments) < 2 {
			return sc.Input, nil
		}
		val, err := lookupNested(sc.Input, segments[1])
		if err != nil {
			// Missing input fields resolve to empty string (supports optional fields).
			return "", nil
		}
		return val, nil

	case "steps":
		if len(segments) < 2 {
			return nil, fmt.Errorf("incomplete step reference: %q", path)
		}
		rest := segments[1]
		// rest is like "create-repo.output.repo_url"
		stepParts := strings.SplitN(rest, ".output.", 2)
		if len(stepParts) != 2 {
			// Could be "create-repo.status"
			stepParts2 := strings.SplitN(rest, ".", 2)
			stepName := stepParts2[0]
			sr, ok := sc.Steps[stepName]
			if !ok {
				return nil, fmt.Errorf("step %q not found", stepName)
			}
			if len(stepParts2) == 2 && stepParts2[1] == "status" {
				return sr.Status, nil
			}
			return nil, fmt.Errorf("invalid step reference: %q", path)
		}
		stepName := stepParts[0]
		outputField := stepParts[1]
		sr, ok := sc.Steps[stepName]
		if !ok {
			return nil, fmt.Errorf("step %q not found", stepName)
		}
		if sr.Output == nil {
			return nil, fmt.Errorf("step %q has no output", stepName)
		}
		return lookupNested(sr.Output, outputField)

	case "env":
		if len(segments) < 2 {
			return nil, fmt.Errorf("incomplete env reference: %q", path)
		}
		val, ok := sc.Env[segments[1]]
		if !ok {
			return "", nil
		}
		return val, nil

	default:
		return nil, fmt.Errorf("unknown variable root %q in %q", root, path)
	}
}

func lookupNested(m map[string]any, path string) (any, error) {
	parts := strings.Split(path, ".")
	var current any = m
	for _, part := range parts {
		mp, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("cannot index into non-object at %q", part)
		}
		current, ok = mp[part]
		if !ok {
			return nil, fmt.Errorf("key %q not found", part)
		}
	}
	return current, nil
}

func applyPipe(val any, fn string) (any, error) {
	s := fmt.Sprintf("%v", val)
	switch fn {
	case "slugify":
		return slugify(s), nil
	case "upper":
		return strings.ToUpper(s), nil
	case "lower":
		return strings.ToLower(s), nil
	case "trim":
		return strings.TrimSpace(s), nil
	default:
		return nil, fmt.Errorf("unknown pipe function %q", fn)
	}
}

func slugify(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else if r == ' ' || r == '-' || r == '_' {
			b.WriteRune('-')
		}
	}
	// Collapse multiple dashes.
	result := b.String()
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}
	return strings.Trim(result, "-")
}
