package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"projects"

	"gopkg.in/yaml.v3"
)

// fieldEdit describes a single field change to apply to the landscape file.
type fieldEdit struct {
	key      string
	newValue string
	isExtra  bool // true if this field belongs under the extra: mapping
	exists   bool // true = update existing field, false = insert new field
}

func main() {
	projectPath := flag.String("project", "", "Path to project.yaml")
	landscapePath := flag.String("landscape", "", "Path to landscape.yml")
	landscapeRepo := flag.String("landscape-repo", "cncf/landscape", "Target repository for the PR (e.g. cncf/landscape)")
	createPR := flag.Bool("create-pr", false, "Create a Pull Request with the changes")
	dryRun := flag.Bool("dry-run", false, "Print changes and PR details without executing")
	flag.Parse()

	if *projectPath == "" || *landscapePath == "" {
		log.Fatal("Both --project and --landscape flags are required")
	}

	// Load Project
	projectData, err := os.ReadFile(*projectPath)
	if err != nil {
		log.Fatalf("Failed to read project file: %v", err)
	}
	var project projects.Project
	if err := yaml.Unmarshal(projectData, &project); err != nil {
		log.Fatalf("Failed to parse project YAML: %v", err)
	}

	// Load Landscape
	landscapeData, err := os.ReadFile(*landscapePath)
	if err != nil {
		log.Fatalf("Failed to read landscape file: %v", err)
	}
	var root yaml.Node
	if err := yaml.Unmarshal(landscapeData, &root); err != nil {
		log.Fatalf("Failed to parse landscape YAML: %v", err)
	}

	lines := strings.Split(string(landscapeData), "\n")

	// Update using line-level edits
	newLines, updated := updateLandscape(&root, &project, lines)
	if !updated {
		log.Printf("No matching entry found or no changes needed for project %s", project.Name)
		os.Exit(0)
	}

	output := strings.Join(newLines, "\n")

	if *dryRun {
		tmpFile, err := os.CreateTemp("", "landscape-*.yml")
		if err != nil {
			log.Fatalf("Failed to create temp file: %v", err)
		}
		defer func() {
			if err := os.Remove(tmpFile.Name()); err != nil {
				log.Printf("Warning: failed to remove temp file: %v", err)
			}
		}()

		if _, err := tmpFile.WriteString(output); err != nil {
			log.Fatalf("Failed to write temp file: %v", err)
		}
		if err := tmpFile.Close(); err != nil {
			log.Fatalf("Failed to close temp file: %v", err)
		}

		cmd := exec.Command("diff", "-u", *landscapePath, tmpFile.Name())
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		fmt.Println("--- Diff ---")
		_ = cmd.Run()

		fmt.Println("\n--- Pull Request Details ---")
		fmt.Printf("Title: Update %s metadata\n", project.Name)
		fmt.Printf("Body: Automated update for %s from cncf/automation\n", project.Name)
		fmt.Printf("Branch: update-%s-%d\n", strings.ReplaceAll(strings.ToLower(project.Name), " ", "-"), time.Now().Unix())
		fmt.Printf("Target Repo: %s\n", *landscapeRepo)
		fmt.Println("Commit will be signed off for DCO compliance")
		return
	}

	// Save
	if err := os.WriteFile(*landscapePath, []byte(output), 0644); err != nil {
		log.Fatalf("Failed to write landscape file: %v", err)
	}
	log.Printf("Successfully updated landscape.yml for project %s", project.Name)

	if *createPR {
		if err := createPullRequest(*landscapePath, *landscapeRepo, &project); err != nil {
			log.Fatalf("Failed to create PR: %v", err)
		}
	}
}

func createPullRequest(landscapePath, landscapeRepo string, project *projects.Project) error {
	absPath, err := filepath.Abs(landscapePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %v", err)
	}
	dir := filepath.Dir(absPath)
	fileName := filepath.Base(absPath)

	// Generate branch name
	safeName := strings.ReplaceAll(strings.ToLower(project.Name), " ", "-")
	branchName := fmt.Sprintf("update-%s-%d", safeName, time.Now().Unix())

	log.Printf("Creating PR for branch %s in %s", branchName, dir)

	// Helper to run git commands
	runGit := func(args ...string) error {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	// Create branch
	if err := runGit("checkout", "-b", branchName); err != nil {
		return fmt.Errorf("git checkout failed: %v", err)
	}

	// Add file
	if err := runGit("add", fileName); err != nil {
		return fmt.Errorf("git add failed: %v", err)
	}

	// Commit with sign-off for DCO compliance
	msg := fmt.Sprintf("Update %s metadata", project.Name)
	if err := runGit("commit", "-s", "-m", msg); err != nil {
		return fmt.Errorf("git commit failed: %v", err)
	}

	// Push
	if err := runGit("push", "origin", branchName); err != nil {
		return fmt.Errorf("git push failed: %v", err)
	}

	// Create PR using gh
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("gh cli not found, cannot create PR")
	}

	prBody := fmt.Sprintf("Automated update for %s from cncf/automation", project.Name)
	cmd := exec.Command("gh", "pr", "create", "--title", msg, "--body", prBody, "--head", branchName, "--repo", landscapeRepo)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh pr create failed: %v", err)
	}

	return nil
}

// updateLandscape navigates the YAML node tree to find the matching project
// entry, then applies line-level edits to the raw file lines.
func updateLandscape(root *yaml.Node, project *projects.Project, lines []string) ([]string, bool) {
	if root.Kind != yaml.DocumentNode {
		return lines, false
	}

	var landscapeSeq *yaml.Node
	if len(root.Content) > 0 && root.Content[0].Kind == yaml.MappingNode {
		for i := 0; i < len(root.Content[0].Content); i += 2 {
			key := root.Content[0].Content[i]
			val := root.Content[0].Content[i+1]
			if key.Value == "landscape" && val.Kind == yaml.SequenceNode {
				landscapeSeq = val
				break
			}
		}
	}

	if landscapeSeq == nil {
		return lines, false
	}

	for _, categoryNode := range landscapeSeq.Content {
		var subcategoriesSeq *yaml.Node
		for i := 0; i < len(categoryNode.Content); i += 2 {
			key := categoryNode.Content[i]
			val := categoryNode.Content[i+1]
			if key.Value == "subcategories" && val.Kind == yaml.SequenceNode {
				subcategoriesSeq = val
				break
			}
		}
		if subcategoriesSeq == nil {
			continue
		}

		for _, subcategoryNode := range subcategoriesSeq.Content {
			var itemsSeq *yaml.Node
			for i := 0; i < len(subcategoryNode.Content); i += 2 {
				key := subcategoryNode.Content[i]
				val := subcategoryNode.Content[i+1]
				if key.Value == "items" && val.Kind == yaml.SequenceNode {
					itemsSeq = val
					break
				}
			}
			if itemsSeq == nil {
				continue
			}

			for _, itemNode := range itemsSeq.Content {
				newLines, matched := matchAndUpdateItem(itemNode, project, lines)
				if matched {
					return newLines, true
				}
			}
		}
	}

	return lines, false
}

// matchAndUpdateItem checks if the given item node matches the project (by name
// and repo_url), and if so, detects changes and applies line-level edits.
func matchAndUpdateItem(itemNode *yaml.Node, project *projects.Project, lines []string) ([]string, bool) {
	var nameNode *yaml.Node
	var repoURLNode *yaml.Node

	for i := 0; i < len(itemNode.Content); i += 2 {
		key := itemNode.Content[i]
		val := itemNode.Content[i+1]
		if key.Value == "name" {
			nameNode = val
		} else if key.Value == "repo_url" {
			repoURLNode = val
		}
	}

	if nameNode == nil || repoURLNode == nil {
		return lines, false
	}

	nameMatch := strings.EqualFold(nameNode.Value, project.Name)
	repoMatch := false
	for _, repo := range project.Repositories {
		if strings.EqualFold(repoURLNode.Value, repo) {
			repoMatch = true
			break
		}
	}

	if !nameMatch || !repoMatch {
		return lines, false
	}

	edits := detectChanges(itemNode, project)
	if len(edits) == 0 {
		return lines, false
	}

	newLines := applyItemEdits(lines, edits, itemNode)
	return newLines, true
}

// detectChanges compares the YAML node tree values against the project and
// returns a list of field edits needed. This is read-only on the node tree.
func detectChanges(itemNode *yaml.Node, project *projects.Project) []fieldEdit {
	var edits []fieldEdit

	checkField := func(key, newValue string, isExtra bool, node *yaml.Node) {
		if newValue == "" {
			return
		}
		found := false
		for i := 0; i < len(node.Content); i += 2 {
			if node.Content[i].Value == key {
				found = true
				if node.Content[i+1].Value != newValue {
					edits = append(edits, fieldEdit{key: key, newValue: newValue, isExtra: isExtra, exists: true})
				}
				break
			}
		}
		if !found {
			edits = append(edits, fieldEdit{key: key, newValue: newValue, isExtra: isExtra, exists: false})
		}
	}

	// Top-level fields
	checkField("homepage_url", project.Website, false, itemNode)
	checkField("description", project.Description, false, itemNode)

	if val, ok := project.Social["twitter"]; ok {
		checkField("twitter", val, false, itemNode)
	}

	// Extra fields
	extraMappings := map[string]string{
		"slack":    "slack_url",
		"linkedin": "linkedin_url",
		"youtube":  "youtube_url",
	}

	needsExtra := false
	for key := range extraMappings {
		if _, ok := project.Social[key]; ok {
			needsExtra = true
			break
		}
	}

	if needsExtra {
		var extraNode *yaml.Node
		for i := 0; i < len(itemNode.Content); i += 2 {
			if itemNode.Content[i].Value == "extra" {
				extraNode = itemNode.Content[i+1]
				break
			}
		}

		for socialKey, landscapeKey := range extraMappings {
			val, ok := project.Social[socialKey]
			if !ok || val == "" {
				continue
			}
			if extraNode != nil {
				checkField(landscapeKey, val, true, extraNode)
			} else {
				edits = append(edits, fieldEdit{key: landscapeKey, newValue: val, isExtra: true, exists: false})
			}
		}
	}

	return edits
}

// applyItemEdits applies the detected field edits to the raw file lines.
// Updates are applied first (they may shrink lines by collapsing block scalars),
// then inserts are applied (they grow lines).
func applyItemEdits(lines []string, edits []fieldEdit, itemNode *yaml.Node) []string {
	result := make([]string, len(lines))
	copy(result, lines)

	start, end := getItemLineRange(result, itemNode)
	fieldIndent := detectFieldIndent(result, start, end)
	extraIndent := fieldIndent + 2

	// Apply updates first (may change line count if collapsing block scalars)
	for _, edit := range edits {
		if !edit.exists {
			continue
		}
		indent := fieldIndent
		if edit.isExtra {
			indent = extraIndent
		}
		result, end = replaceFieldInLines(result, start, end, edit.key, edit.newValue, indent)
	}

	// Apply inserts (each insert shifts subsequent lines)
	for _, edit := range edits {
		if edit.exists {
			continue
		}
		indent := fieldIndent
		if edit.isExtra {
			indent = extraIndent
		}
		result, end = insertFieldInLines(result, start, end, edit.key, edit.newValue, indent, edit.isExtra, fieldIndent)
	}

	return result
}

// getItemLineRange returns the 0-indexed start and end (exclusive) line indices
// for the given item node within the lines slice.
func getItemLineRange(lines []string, itemNode *yaml.Node) (start, end int) {
	start = itemNode.Line - 1 // convert 1-indexed to 0-indexed

	// Detect the indent of the "- " sequence marker on the start line
	dashPos := strings.Index(lines[start], "- ")
	if dashPos < 0 {
		dashPos = 0
	}

	for i := start + 1; i < len(lines); i++ {
		trimmed := strings.TrimLeft(lines[i], " ")
		if trimmed == "" || trimmed == "\r" {
			continue
		}
		indent := len(lines[i]) - len(trimmed)
		if indent <= dashPos {
			return start, i
		}
	}
	return start, len(lines)
}

// detectFieldIndent determines the indentation level for top-level fields
// within an item by examining the lines after the sequence marker line.
func detectFieldIndent(lines []string, start, end int) int {
	for i := start + 1; i < end && i < len(lines); i++ {
		trimmed := strings.TrimLeft(lines[i], " ")
		if trimmed == "" {
			continue
		}
		return len(lines[i]) - len(trimmed)
	}
	// Fallback: start line indent + 2
	trimmed := strings.TrimLeft(lines[start], " ")
	return len(lines[start]) - len(trimmed) + 2
}

// findFieldLine returns the 0-indexed line number where the given key appears
// at the specified indentation, or -1 if not found.
func findFieldLine(lines []string, start, end int, key string, indent int) int {
	prefix := strings.Repeat(" ", indent) + key + ":"
	for i := start; i < end && i < len(lines); i++ {
		if strings.HasPrefix(lines[i], prefix) {
			rest := lines[i][len(prefix):]
			if rest == "" || rest[0] == ' ' || rest[0] == '\r' || rest[0] == '\n' {
				return i
			}
		}
	}
	return -1
}

// findExtraBlockEnd returns the line index (exclusive) where the extra block
// ends. It scans from after the extra: key line for lines indented at or beyond
// extraIndent.
func findExtraBlockEnd(lines []string, extraKeyLine, itemEnd, extraIndent int) int {
	last := extraKeyLine
	for i := extraKeyLine + 1; i < itemEnd && i < len(lines); i++ {
		trimmed := strings.TrimLeft(lines[i], " ")
		if trimmed == "" {
			continue
		}
		lineIndent := len(lines[i]) - len(trimmed)
		if lineIndent >= extraIndent {
			last = i
		} else {
			break
		}
	}
	return last + 1
}

// findLastFieldLine returns the index of the last non-blank line within the
// item that has indentation >= fieldIndent.
func findLastFieldLine(lines []string, start, end, fieldIndent int) int {
	last := start
	for i := start; i < end && i < len(lines); i++ {
		trimmed := strings.TrimLeft(lines[i], " ")
		if trimmed == "" {
			continue
		}
		lineIndent := len(lines[i]) - len(trimmed)
		if lineIndent >= fieldIndent {
			last = i
		}
	}
	return last
}

// replaceFieldInLines replaces the value of an existing field in the raw lines.
// Handles block scalars (>-, >, |, |-) by collapsing continuation lines.
func replaceFieldInLines(lines []string, start, end int, key, newValue string, indent int) ([]string, int) {
	lineIdx := findFieldLine(lines, start, end, key, indent)
	if lineIdx < 0 {
		return lines, end
	}

	prefix := strings.Repeat(" ", indent) + key + ":"
	valuePart := strings.TrimSpace(lines[lineIdx][len(prefix):])

	isBlockScalar := valuePart == ">" || valuePart == ">-" || valuePart == "|" || valuePart == "|-"

	// Replace the key line with the new simple scalar value
	lines[lineIdx] = strings.Repeat(" ", indent) + key + ": " + yamlQuoteIfNeeded(newValue)

	if isBlockScalar {
		// Remove continuation lines (lines indented more than the key)
		contStart := lineIdx + 1
		contEnd := contStart
		for contEnd < end && contEnd < len(lines) {
			trimmed := strings.TrimLeft(lines[contEnd], " ")
			if trimmed == "" {
				break
			}
			lineIndent := len(lines[contEnd]) - len(trimmed)
			if lineIndent > indent {
				contEnd++
			} else {
				break
			}
		}
		if contEnd > contStart {
			lines = append(lines[:contStart], lines[contEnd:]...)
			end -= (contEnd - contStart)
		}
	}

	return lines, end
}

// insertFieldInLines inserts a new field line at the appropriate position.
// For top-level fields, inserts before extra: (if present) or at end of item.
// For extra fields, inserts at end of extra block, creating extra: if needed.
func insertFieldInLines(lines []string, start, end int, key, newValue string, indent int, isExtra bool, fieldIndent int) ([]string, int) {
	newLine := strings.Repeat(" ", indent) + key + ": " + yamlQuoteIfNeeded(newValue)

	var insertAt int

	if isExtra {
		extraLine := findFieldLine(lines, start, end, "extra", fieldIndent)
		if extraLine >= 0 {
			insertAt = findExtraBlockEnd(lines, extraLine, end, fieldIndent+2)
		} else {
			// Create extra: block
			extraKeyLine := strings.Repeat(" ", fieldIndent) + "extra:"
			insertPos := findLastFieldLine(lines, start, end, fieldIndent) + 1
			lines = insertLine(lines, insertPos, extraKeyLine)
			end++
			insertAt = insertPos + 1
		}
	} else {
		extraLine := findFieldLine(lines, start, end, "extra", fieldIndent)
		if extraLine >= 0 {
			insertAt = extraLine
		} else {
			insertAt = findLastFieldLine(lines, start, end, fieldIndent) + 1
		}
	}

	lines = insertLine(lines, insertAt, newLine)
	end++

	return lines, end
}

// insertLine inserts a new line at the given position in the slice.
func insertLine(lines []string, at int, line string) []string {
	lines = append(lines, "")
	copy(lines[at+1:], lines[at:])
	lines[at] = line
	return lines
}

// yamlQuoteIfNeeded returns the value with YAML quoting applied if the value
// contains characters that require it (e.g. ": ", " #", or special starters).
func yamlQuoteIfNeeded(value string) string {
	if value == "" {
		return "''"
	}

	needsQuoting := false

	if strings.Contains(value, ": ") || strings.Contains(value, " #") {
		needsQuoting = true
	}

	specialStarts := "&*!|>'\"%@`{}[],?"
	if strings.ContainsAny(string(value[0]), specialStarts) {
		needsQuoting = true
	}

	if !needsQuoting {
		return value
	}

	escaped := strings.ReplaceAll(value, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	return `"` + escaped + `"`
}
