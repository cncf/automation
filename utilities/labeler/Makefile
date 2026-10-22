# Consumers use the published image from this repo; each repo’s CI passes its own raw
# .../<org>/<repo>/refs/heads/<branch>/.github/labels.yaml. Default here is cncf/automation for local dev.
LABELS_URL ?= https://raw.githubusercontent.com/cncf/automation/refs/heads/main/.github/labels.yaml

run:
	go run main.go $(LABELS_URL) cncf automation 95 "/tag foo"
image:
	docker build -t gha-labeler:latest .