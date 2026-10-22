package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	flag.Parse()

	if len(flag.Args()) < 5 {
		fmt.Println("Usage: labeler [flags] <labels_url> <owner> <repo> <issue_number> <comment_body> <changed_files>")
		os.Exit(1)
	}
	labelsURL := flag.Arg(0)
	owner := flag.Arg(1)
	repo := flag.Arg(2)
	issueNum := flag.Arg(3)
	commentBody := flag.Arg(4)
	changedFiles := flag.Arg(5)

	if labelsURL == "" {
		log.Fatal("labels URL not set")
	}

	cfg, err := LoadConfigFromURL(labelsURL)
	if err != nil {
		log.Fatalf("failed to load labels.yaml: %v", err)
	}

	token := os.Getenv("GITHUB_TOKEN")
	client, err := CreateGitHubClient(token)
	if err != nil {
		log.Fatalf("failed to create GitHub client: %v", err)
	}

	labeler := NewLabeler(client, cfg)

	var files []string
	if changedFiles != "" {
		files = strings.Split(changedFiles, ",")
	}

	issueNumber, err := toInt(issueNum)
	if err != nil {
		log.Fatalf("invalid issue number: %v", err)
	}

	req := &LabelRequest{
		Owner:        owner,
		Repo:         repo,
		IssueNumber:  issueNumber,
		CommentBody:  commentBody,
		ChangedFiles: files,
	}

	ctx := context.Background()
	if err := labeler.ProcessRequest(ctx, req); err != nil {
		log.Fatalf("failed to process request: %v", err)
	}
}

func toInt(s string) (int, error) {
	var i int
	n, err := fmt.Sscanf(s, "%d", &i)
	if err != nil || n != 1 {
		return 0, fmt.Errorf("invalid number: %s", s)
	}
	return i, nil
}
