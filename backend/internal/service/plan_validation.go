package service

import (
	"fmt"
	"strings"

	"gitflame-codepilot/backend/internal/domain"
)

var requiredPlanSections = []string{
	"# Implementation Plan",
	"## Issue Summary",
	"## Goal",
	"## Relevant Files",
	"## Proposed Changes",
	"## Implementation Steps",
	"## Expected Files to Change",
	"## Tests and Verification",
	"## Risks and Open Questions",
}

func ValidatePlan(plan string, repositoryFiles []domain.RepositoryFile) error {
	if strings.TrimSpace(plan) == "" {
		return fmt.Errorf("Agent Engine returned an empty plan")
	}
	position := -1
	for _, section := range requiredPlanSections {
		next := strings.Index(plan, section)
		if next < 0 {
			return fmt.Errorf("plan is missing required section %q", section)
		}
		if next <= position {
			return fmt.Errorf("plan sections are not in the required order")
		}
		position = next
	}
	allowed := make(map[string]struct{}, len(repositoryFiles))
	for _, file := range repositoryFiles {
		allowed[file.Path] = struct{}{}
	}
	relevant := sectionBody(plan, "## Relevant Files", "## Proposed Changes")
	for _, reference := range backtickValues(relevant) {
		if strings.Contains(reference, "/") || strings.Contains(reference, ".") {
			if _, exists := allowed[reference]; !exists && !strings.Contains(relevant, "`"+reference+"` (create)") {
				return fmt.Errorf("plan references repository file %q that was not supplied", reference)
			}
		}
	}
	return nil
}

func sectionBody(text, start, end string) string {
	startIndex := strings.Index(text, start)
	if startIndex < 0 {
		return ""
	}
	endIndex := strings.Index(text[startIndex+len(start):], end)
	if endIndex < 0 {
		return text[startIndex+len(start):]
	}
	return text[startIndex+len(start) : startIndex+len(start)+endIndex]
}

func backtickValues(text string) []string {
	var values []string
	for {
		start := strings.IndexByte(text, '`')
		if start < 0 {
			return values
		}
		text = text[start+1:]
		end := strings.IndexByte(text, '`')
		if end < 0 {
			return values
		}
		values = append(values, text[:end])
		text = text[end+1:]
	}
}
