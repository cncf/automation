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

	// Update
	updated := updateLandscape(&root, &project)
	if !updated {
		log.Printf("No matching entry found for project %s", project.Name)
		os.Exit(0)
	}

	if *dryRun {
		// Create temp file for new content
		tmpFile, err := os.CreateTemp("", "landscape-*.yml")
		if err != nil {
			log.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())

		encoder := yaml.NewEncoder(tmpFile)
		encoder.SetIndent(2)
		if err := encoder.Encode(&root); err != nil {
			log.Fatalf("Failed to encode landscape YAML: %v", err)
		}
		tmpFile.Close()

		// Run diff
		cmd := exec.Command("diff", "-u", *landscapePath, tmpFile.Name())
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		fmt.Println("--- Diff ---")
		_ = cmd.Run() // diff returns exit code 1 if files differ, which is expected

		// Print PR details
		fmt.Println("\n--- Pull Request Details ---")
		fmt.Printf("Title: Update %s metadata\n", project.Name)
		fmt.Printf("Body: Automated update for %s from cncf/automation\n", project.Name)
		fmt.Printf("Branch: update-%s-%d\n", strings.ReplaceAll(strings.ToLower(project.Name), " ", "-"), time.Now().Unix())
		fmt.Printf("Target Repo: %s\n", *landscapeRepo)
		return
	}

	// Save
	f, err := os.Create(*landscapePath)
	if err != nil {
		log.Fatalf("Failed to open landscape file for writing: %v", err)
	}
	defer f.Close()

	encoder := yaml.NewEncoder(f)
	encoder.SetIndent(2)
	if err := encoder.Encode(&root); err != nil {
		log.Fatalf("Failed to encode landscape YAML: %v", err)
	}
	f.Close() // Close explicitly before git operations
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

	// Commit
	msg := fmt.Sprintf("Update %s metadata", project.Name)
	if err := runGit("commit", "-m", msg); err != nil {
		return fmt.Errorf("git commit failed: %v", err)
	}

	// Push
	if err := runGit("push", "origin", branchName); err != nil {
		return fmt.Errorf("git push failed: %v", err)
	}

	// Create PR using gh
	// Check if gh is available
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

func updateLandscape(root *yaml.Node, project *projects.Project) bool {
	// Navigate to 'landscape' key
	if root.Kind != yaml.DocumentNode {
		return false
	}

	var landscapeSeq *yaml.Node

	// Find "landscape" key in the root map
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
		return false
	}

	// Iterate through categories
	for _, categoryNode := range landscapeSeq.Content {
		// Find "subcategories"
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

		// Iterate through subcategories
		for _, subcategoryNode := range subcategoriesSeq.Content {
			// Find "items"
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

			// Iterate through items
			for _, itemNode := range itemsSeq.Content {
				if matchAndUpdateItem(itemNode, project) {
					return true
				}
			}
		}
	}

	return false
}

func matchAndUpdateItem(itemNode *yaml.Node, project *projects.Project) bool {
	var nameNode *yaml.Node
	var repoURLNode *yaml.Node

	// Find name and repo_url
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
		return false
	}

	// Check match
	nameMatch := strings.EqualFold(nameNode.Value, project.Name)
	repoMatch := false
	for _, repo := range project.Repositories {
		if strings.EqualFold(repoURLNode.Value, repo) {
			repoMatch = true
			break
		}
	}

	if nameMatch && repoMatch {
		return updateItemFields(itemNode, project)
	}

	return false
}

func updateItemFields(itemNode *yaml.Node, project *projects.Project) bool {
	changed := false
	// Helper to set or add field
	setField := func(node *yaml.Node, fieldName, value string) {
		if value == "" {
			return
		}
		found := false
		for i := 0; i < len(node.Content); i += 2 {
			key := node.Content[i]
			if key.Value == fieldName {
				if node.Content[i+1].Value != value {
					node.Content[i+1].Value = value
					changed = true
				}
				found = true
				break
			}
		}
		if !found {
			node.Content = append(node.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: fieldName},
				&yaml.Node{Kind: yaml.ScalarNode, Value: value},
			)
			changed = true
		}
	}

	// Helper to get or create 'extra' node
	getExtraNode := func() *yaml.Node {
		for i := 0; i < len(itemNode.Content); i += 2 {
			key := itemNode.Content[i]
			if key.Value == "extra" {
				return itemNode.Content[i+1]
			}
		}
		// Create extra if not found
		extraNode := &yaml.Node{Kind: yaml.MappingNode}
		itemNode.Content = append(itemNode.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "extra"},
			extraNode,
		)
		changed = true
		return extraNode
	}

	setField(itemNode, "homepage_url", project.Website)
	setField(itemNode, "description", project.Description)

	// Top level social fields
	if val, ok := project.Social["twitter"]; ok {
		setField(itemNode, "twitter", val)
	}

	// Extra fields
	extraMappings := map[string]string{
		"slack":    "slack_url",
		"linkedin": "linkedin_url",
		"youtube":  "youtube_url",
	}

	// Check if we need to update extra fields
	needsExtra := false
	for key := range extraMappings {
		if _, ok := project.Social[key]; ok {
			needsExtra = true
			break
		}
	}

	if needsExtra {
		extraNode := getExtraNode()
		for key, landscapeKey := range extraMappings {
			if val, ok := project.Social[key]; ok {
				setField(extraNode, landscapeKey, val)
			}
		}
	}

	return changed
}
