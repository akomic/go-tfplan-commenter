package main

import (
	"os"
	"strings"
	"testing"
)

func TestAnalyzeResourceChanges(t *testing.T) {
	// Test with empty changes
	changes := []ResourceChange{}
	summary := analyzeResourceChanges(changes)

	if len(summary.Create) != 0 || len(summary.Update) != 0 || len(summary.Delete) != 0 || len(summary.Replace) != 0 {
		t.Error("Expected empty summary for empty changes")
	}
}

func TestContainsAction(t *testing.T) {
	actions := []string{"create", "update"}

	if !containsAction(actions, "create") {
		t.Error("Expected to find 'create' action")
	}

	if containsAction(actions, "delete") {
		t.Error("Expected not to find 'delete' action")
	}
}

func TestFormatAttributeValue(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{nil, "(null)"},
		{"", "(empty)"},
		{"test", `"test"`},
		{[]interface{}{}, "[]"},
		{[]interface{}{1, 2, 3}, "[3 items]"},
		{map[string]interface{}{}, "{}"},
		{map[string]interface{}{"key": "value"}, "{1 keys}"},
		{42, "42"},
	}

	for _, test := range tests {
		result := formatAttributeValue(test.input)
		if result != test.expected {
			t.Errorf("formatAttributeValue(%v) = %s, expected %s", test.input, result, test.expected)
		}
	}
}

func TestHasNoChanges(t *testing.T) {
	// Test plan with no changes
	planNoChanges := &TerraformPlan{
		ResourceChanges: []ResourceChange{},
	}
	if !hasNoChanges(planNoChanges) {
		t.Error("Expected hasNoChanges to return true for plan with no resource changes")
	}

	// Test plan with changes
	planWithChanges := &TerraformPlan{
		ResourceChanges: []ResourceChange{
			{
				Address: "aws_s3_bucket.test",
				Change: Change{
					Actions: []string{"create"},
				},
			},
		},
	}
	if hasNoChanges(planWithChanges) {
		t.Error("Expected hasNoChanges to return false for plan with resource changes")
	}
}

func TestFormatResourceList(t *testing.T) {
	// Test empty list
	emptyList := []ResourceDetail{}
	result := formatResourceList(emptyList, 3)
	if result != "" {
		t.Errorf("Expected empty string for empty list, got: %s", result)
	}

	// Test list within limit
	shortList := []ResourceDetail{
		{Address: "aws_s3_bucket.test1"},
		{Address: "aws_s3_bucket.test2"},
	}
	result = formatResourceList(shortList, 3)
	expected := "aws_s3_bucket.test1, aws_s3_bucket.test2"
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}

	// Test list exceeding limit
	longList := []ResourceDetail{
		{Address: "aws_s3_bucket.test1"},
		{Address: "aws_s3_bucket.test2"},
		{Address: "aws_s3_bucket.test3"},
		{Address: "aws_s3_bucket.test4"},
		{Address: "aws_s3_bucket.test5"},
	}
	result = formatResourceList(longList, 3)
	expected = "aws_s3_bucket.test1, aws_s3_bucket.test2, aws_s3_bucket.test3, ... (+2 more)"
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestGenerateMultiPlanMarkdownComment(t *testing.T) {
	// Test with empty plans
	emptyPlans := []PlanInfo{}
	result := generateMultiPlanMarkdownComment(emptyPlans)
	if !strings.Contains(result, "No changes detected across all environments") {
		t.Error("Expected message about no changes for empty plans")
	}

	// Test with plans containing changes
	plans := []PlanInfo{
		{
			Plan: &TerraformPlan{
				TerraformVersion: "1.9.8",
				ResourceChanges: []ResourceChange{
					{
						Address: "aws_s3_bucket.test",
						Change: Change{
							Actions: []string{"create"},
						},
					},
				},
			},
			RelativePath: "env1",
		},
	}
	result = generateMultiPlanMarkdownComment(plans)
	if !strings.Contains(result, "Multi-Environment Terraform Plan Summary") {
		t.Error("Expected multi-environment header")
	}
	if !strings.Contains(result, "env1") {
		t.Error("Expected environment name in output")
	}
	if !strings.Contains(result, "aws_s3_bucket.test") {
		t.Error("Expected resource name in output")
	}
}

func TestShouldSkipAttribute(t *testing.T) {
	skipAttrs := []string{"id", "arn", "tags_all", "timeouts"}
	keepAttrs := []string{"name", "family", "engine", "tags"}

	for _, attr := range skipAttrs {
		if !shouldSkipAttribute(attr) {
			t.Errorf("Expected to skip attribute: %s", attr)
		}
	}

	for _, attr := range keepAttrs {
		if shouldSkipAttribute(attr) {
			t.Errorf("Expected not to skip attribute: %s", attr)
		}
	}
}

func TestGenerateMarkdownComment(t *testing.T) {
	plan := &TerraformPlan{
		TerraformVersion: "1.9.8",
		ResourceChanges:  []ResourceChange{},
	}

	markdown := generateMarkdownComment(plan)

	if !strings.Contains(markdown, "📋 Terraform Plan Summary") {
		t.Error("Expected markdown to contain plan summary header")
	}

	if !strings.Contains(markdown, "No changes detected") {
		t.Error("Expected markdown to indicate no changes for empty plan")
	}

	// Test with actual changes to verify version is included
	planWithChanges := &TerraformPlan{
		TerraformVersion: "1.9.8",
		ResourceChanges: []ResourceChange{
			{
				Address: "test.resource",
				Change: Change{
					Actions: []string{"create"},
					Before:  nil,
					After:   map[string]interface{}{"name": "test"},
				},
			},
		},
	}

	markdownWithChanges := generateMarkdownComment(planWithChanges)
	if !strings.Contains(markdownWithChanges, "1.9.8") {
		t.Error("Expected markdown with changes to contain Terraform version")
	}
}

func TestReadTerraformPlan(t *testing.T) {
	// Test with non-existent file
	_, err := readTerraformPlan("non-existent-file.json")
	if err == nil {
		t.Error("Expected error when reading non-existent file")
	}

	// Test with invalid JSON
	tempFile := "/tmp/invalid.json"
	err = os.WriteFile(tempFile, []byte("invalid json"), 0644)
	if err != nil {
		t.Fatal("Failed to create temp file")
	}
	defer os.Remove(tempFile)

	_, err = readTerraformPlan(tempFile)
	if err == nil {
		t.Error("Expected error when reading invalid JSON")
	}
}
