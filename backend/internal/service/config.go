package service

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"gitflame-codepilot/backend/internal/domain"
)

func ParseAIConfig(raw string) (domain.AIConfig, error) {
	if strings.TrimSpace(raw) == "" {
		return domain.AIConfig{}, errors.New("missing .yml configuration")
	}
	doc := parseSimpleYAML(raw)
	cfg := domain.AIConfig{
		Raw:                raw,
		Version:            scalar(doc, "version", "1"),
		DefaultBranch:      scalar(doc, "repository.default_branch", "main"),
		TargetBranchPrefix: scalar(doc, "repository.target_branch_prefix", "ai/"),
		AnalysisEnabled:    boolean(doc, "analysis.enabled", true),
		RequireApproval:    boolean(doc, "code_generation.require_user_approval", true),
		IncludePatterns:    list(doc, "analysis.include", []string{"**/*"}),
		ExcludePatterns:    list(doc, "analysis.exclude", []string{".git/**", "node_modules/**", "dist/**", "build/**"}),
		MaxFiles:           integer(doc, "analysis.max_files", 20),
		MaxSnippetsPerFile: integer(doc, "analysis.max_snippets_per_file", 3),
		ReviewerPolicy:     scalar(doc, "code_generation.reviewer_policy", "issue_author"),
		ApproveCommand:     scalar(doc, "code_generation.allowed_actions.approve_command", "/approve"),
		CorrectCommand:     scalar(doc, "code_generation.allowed_actions.correct_command", "/correct"),
		RejectCommand:      scalar(doc, "code_generation.allowed_actions.reject_command", "/reject"),
	}
	if cfg.Version != "1" {
		return cfg, fmt.Errorf("unsupported .yml version %q", cfg.Version)
	}
	if !cfg.AnalysisEnabled {
		return cfg, errors.New("repository analysis is disabled in .yml configuration")
	}
	if strings.TrimSpace(cfg.DefaultBranch) == "" || strings.TrimSpace(cfg.TargetBranchPrefix) == "" {
		return cfg, errors.New("repository branch configuration is required")
	}
	if len(cfg.IncludePatterns) == 0 {
		return cfg, errors.New("analysis.include must contain at least one pattern")
	}
	if !cfg.RequireApproval {
		return cfg, errors.New("code_generation.require_user_approval must be true")
	}
	if cfg.ReviewerPolicy != "issue_author" {
		return cfg, fmt.Errorf("unsupported reviewer policy %q", cfg.ReviewerPolicy)
	}
	for name, value := range map[string]string{"approve": cfg.ApproveCommand, "correct": cfg.CorrectCommand, "reject": cfg.RejectCommand} {
		if !strings.HasPrefix(value, "/") {
			return cfg, fmt.Errorf("%s command must start with /", name)
		}
	}
	return cfg, nil
}

type yamlDoc struct {
	scalars map[string]string
	lists   map[string][]string
}

func parseSimpleYAML(raw string) yamlDoc {
	d := yamlDoc{map[string]string{}, map[string][]string{}}
	var stack []string
	current := ""
	for _, line := range strings.Split(raw, "\n") {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		indent := (len(line) - len(strings.TrimLeft(line, " "))) / 2
		if strings.HasPrefix(trim, "- ") {
			if current != "" {
				d.lists[current] = append(d.lists[current], clean(strings.TrimPrefix(trim, "- ")))
			}
			continue
		}
		key, value, ok := strings.Cut(trim, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = clean(value)
		if indent < len(stack) {
			stack = stack[:indent]
		}
		for len(stack) < indent {
			stack = append(stack, "")
		}
		path := strings.Join(append(append([]string{}, stack...), key), ".")
		if value == "" {
			if indent == len(stack) {
				stack = append(stack, key)
			} else {
				stack[indent] = key
			}
			current = path
		} else {
			d.scalars[path] = value
			current = ""
		}
	}
	return d
}
func clean(v string) string {
	v = strings.TrimSpace(v)
	if i := strings.Index(v, " #"); i >= 0 {
		v = v[:i]
	}
	return strings.Trim(strings.TrimSpace(v), `"'`)
}
func scalar(d yamlDoc, k, f string) string {
	if v, ok := d.scalars[k]; ok && v != "" {
		return v
	}
	return f
}
func boolean(d yamlDoc, k string, f bool) bool {
	v, ok := d.scalars[k]
	if !ok {
		return f
	}
	return strings.EqualFold(v, "true")
}
func list(d yamlDoc, k string, f []string) []string {
	if v, ok := d.lists[k]; ok && len(v) > 0 {
		return append([]string(nil), v...)
	}
	return append([]string(nil), f...)
}

func integer(d yamlDoc, key string, fallback int) int {
	value, ok := d.scalars[key]
	if !ok {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return fallback
	}
	return parsed
}
