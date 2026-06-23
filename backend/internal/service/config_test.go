package service

import "testing"

func TestParseAIConfigRejectsInvalidYAMLAndModelSelection(t *testing.T) {
	if _, err := ParseAIConfig("analysis: ["); err == nil {
		t.Fatal("expected malformed YAML to be rejected")
	}
	if _, err := ParseAIConfig("version: 1\nagent_model: attacker/model\n"); err == nil {
		t.Fatal("expected repository-controlled model selection to be rejected")
	}
	if _, err := ParseAIConfig("version: 1\nrecommendations:\n  retention_days: nope\n"); err == nil {
		t.Fatal("expected invalid retention period to be rejected")
	}
}

func TestParseAIConfigSupportsInlineLists(t *testing.T) {
	cfg, err := ParseAIConfig("version: 1\nanalysis:\n  enabled: true\n  include: [internal/**, cmd/**]\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.IncludePatterns) != 2 || cfg.IncludePatterns[0] != "internal/**" || cfg.IncludePatterns[1] != "cmd/**" {
		t.Fatalf("unexpected inline include patterns: %#v", cfg.IncludePatterns)
	}
}
