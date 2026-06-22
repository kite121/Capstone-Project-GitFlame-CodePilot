package service

import (
	"fmt"
	"path"
	"regexp"
	"strings"

	"gitflame-codepilot/backend/internal/domain"
)

var requiredPlanSections = []string{
	"Implementation Plan", "Issue Summary", "Goal", "Relevant Files", "Proposed Changes",
	"Implementation Steps", "Expected Files to Change", "Tests and Verification",
	"Risks and Open Questions",
}

var headingPattern = regexp.MustCompile(`(?m)^(#{1,6})\s+(.+?)\s*$`)
var fileBulletPattern = regexp.MustCompile("(?m)^\\s*[-*]\\s+`([^`]+)`(\\s*\\(create\\))?\\s*:\\s*(.+?)\\s*$")
var orderedStepPattern = regexp.MustCompile(`(?m)^\s*1\.\s+\S`)

func ValidatePlan(markdown string, repositoryFiles []domain.RepositoryFile) error {
	plan := strings.TrimSpace(markdown)
	if plan == "" {
		return fmt.Errorf("Agent Engine returned an empty plan")
	}
	if len(plan) > 200_000 {
		return fmt.Errorf("plan exceeds 200000 characters")
	}
	if strings.Contains(plan, "```") {
		return fmt.Errorf("fenced code blocks are not allowed")
	}
	if !strings.HasPrefix(plan, "# Implementation Plan\n") {
		return fmt.Errorf("plan must start with # Implementation Plan")
	}
	headings := headingPattern.FindAllStringSubmatch(plan, -1)
	if len(headings) != len(requiredPlanSections) {
		return fmt.Errorf("headings are missing, duplicated, or out of order")
	}
	for index, expected := range requiredPlanSections {
		level := "##"
		if index == 0 {
			level = "#"
		}
		if headings[index][1] != level || headings[index][2] != expected {
			return fmt.Errorf("headings are missing, duplicated, or out of order")
		}
	}
	sections := splitPlanSections(plan, headings)
	for _, name := range requiredPlanSections[1:] {
		if strings.TrimSpace(sections[name]) == "" {
			return fmt.Errorf("section %s is empty", name)
		}
	}
	if !orderedStepPattern.MatchString(sections["Implementation Steps"]) {
		return fmt.Errorf("Implementation Steps must contain an ordered list")
	}
	allowed := make(map[string]struct{}, len(repositoryFiles))
	for _, file := range repositoryFiles {
		allowed[normalizePlanPath(file.Path)] = struct{}{}
	}
	for _, section := range []string{"Relevant Files", "Expected Files to Change"} {
		bullets := fileBulletPattern.FindAllStringSubmatch(sections[section], -1)
		if len(bullets) == 0 {
			return fmt.Errorf("section %s must contain path bullets", section)
		}
		for _, bullet := range bullets {
			filePath := normalizePlanPath(bullet[1])
			if !safePlanPath(filePath) {
				return fmt.Errorf("plan contains unsafe repository path: %s", bullet[1])
			}
			isCreate := strings.TrimSpace(bullet[2]) != ""
			if _, exists := allowed[filePath]; !exists && !isCreate {
				return fmt.Errorf("plan references unavailable repository file: %s", filePath)
			}
		}
	}
	return nil
}

func splitPlanSections(plan string, headings [][]string) map[string]string {
	positions := headingPattern.FindAllStringSubmatchIndex(plan, -1)
	result := make(map[string]string, len(positions))
	for index, position := range positions {
		start := position[1]
		end := len(plan)
		if index+1 < len(positions) {
			end = positions[index+1][0]
		}
		result[headings[index][2]] = strings.TrimSpace(plan[start:end])
	}
	return result
}

func normalizePlanPath(value string) string {
	return strings.TrimPrefix(path.Clean(strings.ReplaceAll(value, "\\", "/")), "./")
}
func safePlanPath(value string) bool {
	return value != "" && value != "." && !strings.HasPrefix(value, "/") && value != ".." && !strings.HasPrefix(value, "../") && value != ".git" && !strings.HasPrefix(value, ".git/")
}
