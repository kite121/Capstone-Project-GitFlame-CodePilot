package service

import (
	"strings"
	"testing"

	"gitflame-codepilot/backend/internal/domain"
)

func TestValidatePlan(t *testing.T) {
	plan := validPlan("internal/httpapi/server.go")
	if err := ValidatePlan(plan, []domain.RepositoryFile{{Path: "internal/httpapi/server.go"}}); err != nil {
		t.Fatal(err)
	}
	if err := ValidatePlan(strings.Replace(plan, "## Goal", "## Objective", 1), []domain.RepositoryFile{{Path: "internal/httpapi/server.go"}}); err == nil {
		t.Fatal("expected missing section error")
	}
	if err := ValidatePlan(validPlan("invented/file.go"), []domain.RepositoryFile{{Path: "internal/httpapi/server.go"}}); err == nil {
		t.Fatal("expected hallucinated file error")
	}
}

func validPlan(path string) string {
	return `# Implementation Plan

## Issue Summary
Summary.

## Goal
Goal.

## Relevant Files
- ` + "`" + path + "`" + `: relevant.

## Proposed Changes
- Change behavior.

## Implementation Steps
1. Implement.

## Expected Files to Change
- ` + "`" + path + "`" + `: modify.

## Tests and Verification
- Verify.

## Risks and Open Questions
- TBD.
`
}
