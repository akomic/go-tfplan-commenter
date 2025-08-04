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
