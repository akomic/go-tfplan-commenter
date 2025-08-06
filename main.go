package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Version information - will be set during build
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

// TerraformPlan represents the structure of a Terraform plan JSON
type TerraformPlan struct {
	FormatVersion    string           `json:"format_version"`
	TerraformVersion string           `json:"terraform_version"`
	ResourceChanges  []ResourceChange `json:"resource_changes"`
}

// PlanInfo holds a plan with its relative path information
type PlanInfo struct {
	Plan         *TerraformPlan
	RelativePath string // e.g., "env1/dev" for ./tfplans/env1/dev/tfplan.json
}

// ResourceChange represents a single resource change in the plan
type ResourceChange struct {
	Address       string `json:"address"`
	ModuleAddress string `json:"module_address"`
	Mode          string `json:"mode"`
	Type          string `json:"type"`
	Name          string `json:"name"`
	ProviderName  string `json:"provider_name"`
	Change        Change `json:"change"`
}

// Change represents the actual change being made to a resource
type Change struct {
	Actions      []string    `json:"actions"`
	Before       interface{} `json:"before"`
	After        interface{} `json:"after"`
	AfterUnknown interface{} `json:"after_unknown"`
}

// ResourceSummary holds the summary of changes for each action type
type ResourceSummary struct {
	Create  []ResourceDetail
	Update  []ResourceDetail
	Delete  []ResourceDetail
	Replace []ResourceDetail
}

// ResourceDetail holds detailed information about a resource change
type ResourceDetail struct {
	Address     string
	Changes     []AttributeChange
	ForceReason string // For resources being deleted/replaced
}

// AttributeChange represents a change to a specific attribute
type AttributeChange struct {
	Attribute string
	Before    interface{}
	After     interface{}
	IsNew     bool
	IsRemoved bool
}

func main() {
	var showVersion = flag.Bool("version", false, "Show version information")
	var showHelp = flag.Bool("help", false, "Show help information")
	flag.Parse()

	if *showVersion {
		fmt.Printf("tfplan-commenter version %s\n", Version)
		fmt.Printf("Git commit: %s\n", GitCommit)
		fmt.Printf("Build date: %s\n", BuildDate)
		os.Exit(0)
	}

	if *showHelp {
		printUsage()
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) < 1 {
		printUsage()
		os.Exit(1)
	}

	inputPath := args[0]
	outputFile := "terraform-plan-comment.md"
	if len(args) > 1 {
		outputFile = args[1]
	}

	// Check if input is a file or directory
	fileInfo, err := os.Stat(inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error accessing input path: %v\n", err)
		os.Exit(1)
	}

	var plans []PlanInfo
	var markdown string

	if fileInfo.IsDir() {
		// Process directory containing multiple plan files
		plans, err = findAndReadPlanFiles(inputPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error processing directory: %v\n", err)
			os.Exit(1)
		}

		if len(plans) == 0 {
			fmt.Fprintf(os.Stderr, "No tfplan.json files found in directory: %s\n", inputPath)
			os.Exit(1)
		}

		markdown = generateMultiPlanMarkdownComment(plans)
	} else {
		// Process single plan file
		plan, err := readTerraformPlan(inputPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading plan file: %v\n", err)
			os.Exit(1)
		}

		markdown = generateMarkdownComment(plan)
	}

	// Write to output file
	err = os.WriteFile(outputFile, []byte(markdown), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
		os.Exit(1)
	}

	if fileInfo.IsDir() {
		fmt.Printf("Multi-plan comment generated from %d plan(s): %s\n", len(plans), outputFile)
	} else {
		fmt.Printf("Terraform plan comment generated: %s\n", outputFile)
	}
}

func printUsage() {
	fmt.Printf("tfplan-commenter version %s\n\n", Version)
	fmt.Println("Usage: tfplan-commenter [options] <input> [output.md]")
	fmt.Println()
	fmt.Println("Arguments:")
	fmt.Println("  input        Path to a Terraform plan JSON file or directory containing tfplan.json files")
	fmt.Println("  output.md    Output markdown file (default: terraform-plan-comment.md)")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -version     Show version information")
	fmt.Println("  -help        Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Process single plan file")
	fmt.Println("  tfplan-commenter tfplan.json")
	fmt.Println("  tfplan-commenter tfplan.json my-comment.md")
	fmt.Println()
	fmt.Println("  # Process directory with multiple plan files")
	fmt.Println("  tfplan-commenter ./tfplans/")
	fmt.Println("  tfplan-commenter ./environments/ multi-env-comment.md")
	fmt.Println()
	fmt.Println("  # Show version")
	fmt.Println("  tfplan-commenter -version")
	fmt.Println()
	fmt.Println("Directory Processing:")
	fmt.Println("  When processing a directory, the tool will:")
	fmt.Println("  - Recursively search for 'tfplan.json' files")
	fmt.Println("  - Skip plans with no changes")
	fmt.Println("  - Group results by relative path (e.g., 'env1/dev' for ./tfplans/env1/dev/tfplan.json)")
	fmt.Println("  - Generate a single markdown comment with all plans")
}

func findAndReadPlanFiles(rootDir string) ([]PlanInfo, error) {
	var plans []PlanInfo

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Look for files named "tfplan.json"
		if !info.IsDir() && info.Name() == "tfplan.json" {
			// Read and parse the plan file
			plan, err := readTerraformPlan(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to read plan file %s: %v\n", path, err)
				return nil // Continue processing other files
			}

			// Skip plans with no changes
			if hasNoChanges(plan) {
				fmt.Printf("Skipping %s (no changes)\n", path)
				return nil
			}

			// Calculate relative path from root directory
			relPath, err := filepath.Rel(rootDir, filepath.Dir(path))
			if err != nil {
				relPath = filepath.Dir(path)
			}

			// Clean up the relative path (remove leading ./ if present)
			if relPath == "." {
				relPath = "root"
			}

			plans = append(plans, PlanInfo{
				Plan:         plan,
				RelativePath: relPath,
			})

			fmt.Printf("Found plan with changes: %s\n", path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking directory: %w", err)
	}

	// Sort plans by relative path for consistent output
	sort.Slice(plans, func(i, j int) bool {
		return plans[i].RelativePath < plans[j].RelativePath
	})

	return plans, nil
}

func hasNoChanges(plan *TerraformPlan) bool {
	return len(plan.ResourceChanges) == 0
}

func readTerraformPlan(filename string) (*TerraformPlan, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var plan TerraformPlan
	err = json.Unmarshal(data, &plan)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &plan, nil
}

func generateMultiPlanMarkdownComment(plans []PlanInfo) string {
	var md strings.Builder

	// Header
	md.WriteString("## 📋 Multi-Environment Terraform Plan Summary\n\n")

	// Overall statistics across all plans
	totalCreate, totalUpdate, totalDelete, totalReplace := 0, 0, 0, 0
	var allTerraformVersions []string

	for _, planInfo := range plans {
		summary := analyzeResourceChanges(planInfo.Plan.ResourceChanges)
		totalCreate += len(summary.Create)
		totalUpdate += len(summary.Update)
		totalDelete += len(summary.Delete)
		totalReplace += len(summary.Replace)

		// Collect unique Terraform versions
		version := planInfo.Plan.TerraformVersion
		found := false
		for _, v := range allTerraformVersions {
			if v == version {
				found = true
				break
			}
		}
		if !found {
			allTerraformVersions = append(allTerraformVersions, version)
		}
	}

	totalChanges := totalCreate + totalUpdate + totalDelete + totalReplace

	if totalChanges == 0 {
		md.WriteString("✅ **No changes detected across all environments** - Infrastructure is up to date!\n\n")
		return md.String()
	}

	md.WriteString(fmt.Sprintf("**Environments processed:** %d\n", len(plans)))
	md.WriteString(fmt.Sprintf("**Total resources affected:** %d\n\n", totalChanges))

	// Overall summary table
	md.WriteString("### 📊 Overall Summary\n\n")
	md.WriteString("| Action | Total Count |\n")
	md.WriteString("|--------|-------------|\n")

	if totalCreate > 0 {
		md.WriteString(fmt.Sprintf("| 🟢 **Create** | %d |\n", totalCreate))
	}
	if totalUpdate > 0 {
		md.WriteString(fmt.Sprintf("| 🟡 **Update** | %d |\n", totalUpdate))
	}
	if totalReplace > 0 {
		md.WriteString(fmt.Sprintf("| 🔄 **Replace** | %d |\n", totalReplace))
	}
	if totalDelete > 0 {
		md.WriteString(fmt.Sprintf("| 🔴 **Delete** | %d |\n", totalDelete))
	}

	md.WriteString("\n")

	// Environment-specific sections
	md.WriteString("### 🏗️ Environment Details\n\n")

	for _, planInfo := range plans {
		summary := analyzeResourceChanges(planInfo.Plan.ResourceChanges)
		envTotalChanges := len(summary.Create) + len(summary.Update) + len(summary.Delete) + len(summary.Replace)

		// Environment header
		md.WriteString(fmt.Sprintf("#### 📁 `%s`\n\n", planInfo.RelativePath))

		if envTotalChanges == 0 {
			md.WriteString("✅ No changes in this environment\n\n")
			continue
		}

		// Environment summary table
		md.WriteString("| Action | Count | Resources |\n")
		md.WriteString("|--------|-------|----------|\n")

		if len(summary.Create) > 0 {
			md.WriteString(fmt.Sprintf("| 🟢 **Create** | %d | %s |\n",
				len(summary.Create),
				formatResourceList(summary.Create, 3)))
		}

		if len(summary.Update) > 0 {
			md.WriteString(fmt.Sprintf("| 🟡 **Update** | %d | %s |\n",
				len(summary.Update),
				formatResourceList(summary.Update, 3)))
		}

		if len(summary.Replace) > 0 {
			md.WriteString(fmt.Sprintf("| 🔄 **Replace** | %d | %s |\n",
				len(summary.Replace),
				formatResourceList(summary.Replace, 3)))
		}

		if len(summary.Delete) > 0 {
			md.WriteString(fmt.Sprintf("| 🔴 **Delete** | %d | %s |\n",
				len(summary.Delete),
				formatResourceList(summary.Delete, 3)))
		}

		md.WriteString("\n")

		// Detailed sections for this environment
		if len(summary.Create) > 0 {
			md.WriteString("**🟢 Resources to be Created:**\n")
			for _, resource := range summary.Create {
				md.WriteString(fmt.Sprintf("- `%s`\n", resource.Address))
			}
			md.WriteString("\n")
		}

		if len(summary.Update) > 0 {
			md.WriteString("**🟡 Resources to be Updated:**\n")
			for _, resource := range summary.Update {
				md.WriteString(fmt.Sprintf("- `%s`", resource.Address))
				if len(resource.Changes) > 0 {
					md.WriteString(" - ")
					var changeDescs []string
					for _, change := range resource.Changes {
						if change.IsNew {
							changeDescs = append(changeDescs, fmt.Sprintf("%s *(new)*", change.Attribute))
						} else if change.IsRemoved {
							changeDescs = append(changeDescs, fmt.Sprintf("%s *(removed)*", change.Attribute))
						} else {
							changeDescs = append(changeDescs, change.Attribute)
						}
					}
					md.WriteString(strings.Join(changeDescs, ", "))
				}
				md.WriteString("\n")
			}
			md.WriteString("\n")
		}

		if len(summary.Replace) > 0 {
			md.WriteString("**🔄 Resources to be Replaced:**\n")
			for _, resource := range summary.Replace {
				md.WriteString(fmt.Sprintf("- `%s`", resource.Address))
				if resource.ForceReason != "" {
					md.WriteString(fmt.Sprintf(" - %s", resource.ForceReason))
				}
				md.WriteString("\n")
			}
			md.WriteString("\n")
		}

		if len(summary.Delete) > 0 {
			md.WriteString("**🔴 Resources to be Deleted:**\n")
			for _, resource := range summary.Delete {
				md.WriteString(fmt.Sprintf("- `%s`", resource.Address))
				if resource.ForceReason != "" {
					md.WriteString(fmt.Sprintf(" - %s", resource.ForceReason))
				}
				md.WriteString("\n")
			}
			md.WriteString("\n")
		}

		md.WriteString("---\n\n")
	}

	// Footer
	if len(allTerraformVersions) == 1 {
		md.WriteString(fmt.Sprintf("*Generated from Terraform %s plans*\n", allTerraformVersions[0]))
	} else {
		md.WriteString(fmt.Sprintf("*Generated from Terraform plans (versions: %s)*\n", strings.Join(allTerraformVersions, ", ")))
	}

	return md.String()
}

func generateMarkdownComment(plan *TerraformPlan) string {
	summary := analyzeResourceChanges(plan.ResourceChanges)

	var md strings.Builder

	// Header
	md.WriteString("## 📋 Terraform Plan Summary\n\n")

	// Overall statistics
	totalChanges := len(summary.Create) + len(summary.Update) + len(summary.Delete) + len(summary.Replace)

	if totalChanges == 0 {
		md.WriteString("✅ **No changes detected** - Infrastructure is up to date!\n\n")
		return md.String()
	}

	md.WriteString(fmt.Sprintf("**Total resources affected:** %d\n\n", totalChanges))

	// Summary table
	md.WriteString("| Action | Count | Resources |\n")
	md.WriteString("|--------|-------|----------|\n")

	if len(summary.Create) > 0 {
		md.WriteString(fmt.Sprintf("| 🟢 **Create** | %d | %s |\n",
			len(summary.Create),
			formatResourceList(summary.Create, 3)))
	}

	if len(summary.Update) > 0 {
		md.WriteString(fmt.Sprintf("| 🟡 **Update** | %d | %s |\n",
			len(summary.Update),
			formatResourceList(summary.Update, 3)))
	}

	if len(summary.Replace) > 0 {
		md.WriteString(fmt.Sprintf("| 🔄 **Replace** | %d | %s |\n",
			len(summary.Replace),
			formatResourceList(summary.Replace, 3)))
	}

	if len(summary.Delete) > 0 {
		md.WriteString(fmt.Sprintf("| 🔴 **Delete** | %d | %s |\n",
			len(summary.Delete),
			formatResourceList(summary.Delete, 3)))
	}

	md.WriteString("\n")

	// Detailed sections for each action type
	if len(summary.Create) > 0 {
		md.WriteString("### 🟢 Resources to be Created\n\n")
		for _, resource := range summary.Create {
			md.WriteString(fmt.Sprintf("- `%s`\n", resource.Address))
		}
		md.WriteString("\n")
	}

	if len(summary.Update) > 0 {
		md.WriteString("### 🟡 Resources to be Updated\n\n")
		for _, resource := range summary.Update {
			md.WriteString(fmt.Sprintf("#### `%s`\n\n", resource.Address))
			if len(resource.Changes) > 0 {
				md.WriteString("**Attributes being modified:**\n\n")
				for _, change := range resource.Changes {
					if change.IsNew {
						md.WriteString(fmt.Sprintf("- **%s**: %s *(new)*\n",
							change.Attribute, formatAttributeValue(change.After)))
					} else if change.IsRemoved {
						md.WriteString(fmt.Sprintf("- **%s**: %s *(removed)*\n",
							change.Attribute, formatAttributeValue(change.Before)))
					} else {
						md.WriteString(fmt.Sprintf("- **%s**: %s → %s\n",
							change.Attribute,
							formatAttributeValue(change.Before),
							formatAttributeValue(change.After)))
					}
				}
			} else {
				md.WriteString("*No specific attribute changes detected*\n")
			}
			md.WriteString("\n")
		}
	}

	if len(summary.Replace) > 0 {
		md.WriteString("### 🔄 Resources to be Replaced\n\n")
		for _, resource := range summary.Replace {
			md.WriteString(fmt.Sprintf("#### `%s`\n\n", resource.Address))
			if resource.ForceReason != "" {
				md.WriteString(fmt.Sprintf("**Reason for replacement:** %s\n\n", resource.ForceReason))
			}
			if len(resource.Changes) > 0 {
				md.WriteString("**Attribute changes:**\n\n")
				for _, change := range resource.Changes {
					if change.IsNew {
						md.WriteString(fmt.Sprintf("- **%s**: %s *(new)*\n",
							change.Attribute, formatAttributeValue(change.After)))
					} else if change.IsRemoved {
						md.WriteString(fmt.Sprintf("- **%s**: %s *(removed)*\n",
							change.Attribute, formatAttributeValue(change.Before)))
					} else {
						md.WriteString(fmt.Sprintf("- **%s**: %s → %s\n",
							change.Attribute,
							formatAttributeValue(change.Before),
							formatAttributeValue(change.After)))
					}
				}
			}
			md.WriteString("\n")
		}
	}

	if len(summary.Delete) > 0 {
		md.WriteString("### 🔴 Resources to be Deleted\n\n")
		for _, resource := range summary.Delete {
			md.WriteString(fmt.Sprintf("#### `%s`\n\n", resource.Address))
			if resource.ForceReason != "" {
				md.WriteString(fmt.Sprintf("**Resource details:** %s\n\n", resource.ForceReason))
			}
		}
	}

	// Footer
	md.WriteString("---\n")
	md.WriteString(fmt.Sprintf("*Generated from Terraform %s plan*\n", plan.TerraformVersion))

	return md.String()
}

func analyzeResourceChanges(changes []ResourceChange) ResourceSummary {
	summary := ResourceSummary{
		Create:  make([]ResourceDetail, 0),
		Update:  make([]ResourceDetail, 0),
		Delete:  make([]ResourceDetail, 0),
		Replace: make([]ResourceDetail, 0),
	}

	for _, change := range changes {
		resourceName := change.Address
		actions := change.Change.Actions

		detail := ResourceDetail{
			Address: resourceName,
			Changes: analyzeAttributeChanges(change.Change),
		}

		// Determine the primary action
		if containsAction(actions, "create") && containsAction(actions, "delete") {
			// This is a replace operation
			detail.ForceReason = determineReplaceReason(change.Change)
			summary.Replace = append(summary.Replace, detail)
		} else if containsAction(actions, "create") {
			summary.Create = append(summary.Create, detail)
		} else if containsAction(actions, "update") {
			summary.Update = append(summary.Update, detail)
		} else if containsAction(actions, "delete") {
			detail.ForceReason = determineDeleteReason(change.Change)
			summary.Delete = append(summary.Delete, detail)
		}
	}

	// Sort all slices for consistent output
	sortResourceDetails(summary.Create)
	sortResourceDetails(summary.Update)
	sortResourceDetails(summary.Delete)
	sortResourceDetails(summary.Replace)

	return summary
}

func containsAction(actions []string, action string) bool {
	for _, a := range actions {
		if a == action {
			return true
		}
	}
	return false
}

func formatResourceList(resources []ResourceDetail, maxDisplay int) string {
	if len(resources) == 0 {
		return ""
	}

	resourceNames := make([]string, len(resources))
	for i, r := range resources {
		resourceNames[i] = r.Address
	}

	if len(resourceNames) <= maxDisplay {
		return strings.Join(resourceNames, ", ")
	}

	displayed := resourceNames[:maxDisplay]
	remaining := len(resourceNames) - maxDisplay

	return fmt.Sprintf("%s, ... (+%d more)", strings.Join(displayed, ", "), remaining)
}

func analyzeAttributeChanges(change Change) []AttributeChange {
	var changes []AttributeChange

	beforeMap, beforeOk := change.Before.(map[string]interface{})
	afterMap, afterOk := change.After.(map[string]interface{})

	if !beforeOk || !afterOk {
		return changes
	}

	// Find all unique keys
	allKeys := make(map[string]bool)
	for key := range beforeMap {
		allKeys[key] = true
	}
	for key := range afterMap {
		allKeys[key] = true
	}

	// Analyze each attribute
	for key := range allKeys {
		beforeVal, beforeExists := beforeMap[key]
		afterVal, afterExists := afterMap[key]

		// Skip certain system attributes that are not meaningful to users
		if shouldSkipAttribute(key) {
			continue
		}

		if !beforeExists && afterExists {
			// New attribute
			changes = append(changes, AttributeChange{
				Attribute: key,
				Before:    nil,
				After:     afterVal,
				IsNew:     true,
			})
		} else if beforeExists && !afterExists {
			// Removed attribute
			changes = append(changes, AttributeChange{
				Attribute: key,
				Before:    beforeVal,
				After:     nil,
				IsRemoved: true,
			})
		} else if beforeExists && afterExists && !deepEqual(beforeVal, afterVal) {
			// Changed attribute
			changes = append(changes, AttributeChange{
				Attribute: key,
				Before:    beforeVal,
				After:     afterVal,
			})
		}
	}

	return changes
}

func shouldSkipAttribute(key string) bool {
	skipAttributes := []string{
		"id", "arn", "tags_all", "timeouts",
	}

	for _, skip := range skipAttributes {
		if key == skip {
			return true
		}
	}
	return false
}

func deepEqual(a, b interface{}) bool {
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func determineReplaceReason(change Change) string {
	changes := analyzeAttributeChanges(change)

	// Look for attributes that commonly force replacement
	forceReplaceAttrs := []string{"name", "family", "engine", "vpc_id", "availability_zone"}

	for _, attrChange := range changes {
		for _, forceAttr := range forceReplaceAttrs {
			if attrChange.Attribute == forceAttr {
				return fmt.Sprintf("Attribute '%s' changed from '%v' to '%v' (forces replacement)",
					attrChange.Attribute, attrChange.Before, attrChange.After)
			}
		}
	}

	if len(changes) > 0 {
		return fmt.Sprintf("Multiple attribute changes require replacement")
	}

	return "Resource configuration requires replacement"
}

func determineDeleteReason(change Change) string {
	// For delete operations, we mainly care about what's being removed
	beforeMap, ok := change.Before.(map[string]interface{})
	if !ok {
		return "Resource marked for deletion"
	}

	// Identify key attributes that might explain why it's being deleted
	keyAttrs := []string{"name", "id", "family", "engine"}
	var identifiers []string

	for _, attr := range keyAttrs {
		if val, exists := beforeMap[attr]; exists && val != nil {
			identifiers = append(identifiers, fmt.Sprintf("%s: %v", attr, val))
		}
	}

	if len(identifiers) > 0 {
		return fmt.Sprintf("Resource with %s", strings.Join(identifiers, ", "))
	}

	return "Resource marked for deletion"
}

func sortResourceDetails(resources []ResourceDetail) {
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].Address < resources[j].Address
	})
}

func formatAttributeValue(val interface{}) string {
	if val == nil {
		return "(null)"
	}

	switch v := val.(type) {
	case string:
		if v == "" {
			return "(empty)"
		}
		return fmt.Sprintf(`"%s"`, v)
	case []interface{}:
		if len(v) == 0 {
			return "[]"
		}
		return fmt.Sprintf("[%d items]", len(v))
	case map[string]interface{}:
		if len(v) == 0 {
			return "{}"
		}
		return fmt.Sprintf("{%d keys}", len(v))
	default:
		return fmt.Sprintf("%v", v)
	}
}
