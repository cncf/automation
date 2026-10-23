package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"projects"
)

func main() {
	var (
		configFile          = flag.String("config", "yaml/projectlist.yaml", "Path to project list configuration file")
		repoRoot            = flag.String("repo-root", "", "Root of a .project repository; discovers and validates every project it contains (overrides -config and -maintainers)")
		cacheDir            = flag.String("cache", ".cache", "Directory to store cached validation results")
		maintainersFile     = flag.String("maintainers", "yaml/maintainers.yaml", "Path to maintainers file (set empty to skip)")
		baseMaintainersFile = flag.String("base-maintainers", "", "Path to base maintainers file, or to a base repository root, for diff validation")
		verifyMaintainers   = flag.Bool("verify-maintainers", false, "Verify maintainer handles via external service (stubbed)")
		outputFormat        = flag.String("output", "text", "Output format: text, json, yaml")
		checkLinks          = flag.Bool("check-links", false, "Check that every link and maintainer handle resolves; 404/410 fails the run. On by default with -committees")
		linksBase           = flag.String("links-base", "", "Base repository root; links already present there are not checked, so a pull request only answers for the links it adds")
		linksReport         = flag.String("links-report", "", "Write a markdown report of broken links here (an empty file means none were found)")
		envFile             = flag.String("env-file", ".env", "Env file read for GITHUB_TOKEN when checking links; real env vars take precedence, and 'gh auth token' is the fallback")
		committeesFile      = flag.String("committees", "", "Path to a committees.yaml file; validates only that file and exits. Checks every handle and URL unless -check-links=false (-output, -links-base, -links-report and -env-file apply)")
	)
	flag.Parse()

	if *committeesFile != "" {
		// committees.yaml is small and every entry is a GitHub handle, so
		// link checking is on unless explicitly disabled.
		committeeLinks := true
		flag.Visit(func(f *flag.Flag) {
			if f.Name == "check-links" {
				committeeLinks = *checkLinks
			}
		})
		var links *linkOptions
		if committeeLinks {
			links = &linkOptions{base: *linksBase, report: *linksReport, token: githubToken(*envFile)}
		}
		os.Exit(runCommittees(*committeesFile, *outputFormat, links))
	}

	// The composite actions each cover one half of the repository, so
	// repo-root mode has to honour the established "skip" idioms: an explicit
	// -maintainers "" means projects only, and an explicit -config /dev/null
	// means maintainers only. Without this, running both actions on a
	// multi-project repo would report every file twice.
	skipProjects, skipMaintainers := false, false
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "config":
			v := f.Value.String()
			skipProjects = v == "" || v == "/dev/null"
		case "maintainers":
			skipMaintainers = f.Value.String() == ""
		}
	})

	// In repo-root mode the layout decides which files to validate, so the
	// explicit -config and -maintainers paths are replaced by discovery.
	var discovery *projects.Discovery
	if *repoRoot != "" {
		d, err := projects.Discover(*repoRoot)
		if err != nil {
			log.Fatalf("discovery failed: %v", err)
		}
		discovery = d

		*configFile = ""
		if !skipProjects {
			listFile, cleanup, err := writeProjectList(d)
			if err != nil {
				log.Fatalf("failed to build project list: %v", err)
			}
			defer cleanup()
			*configFile = listFile
		}

		*maintainersFile = ""
	}

	if *configFile == "" {
		// Create a temporary dummy config file if none provided
		f, err := os.CreateTemp("", "dummy-projectlist-*.yaml")
		if err != nil {
			log.Fatal("failed to create temporary config file")
		}
		f.WriteString("projects: []")
		f.Close()
		defer os.Remove(f.Name())
		*configFile = f.Name()
	}

	validator := projects.NewValidator(*cacheDir)

	// Repository-wide checks run first: when the layout itself is wrong, the
	// per-file results are reported against files that may not be the ones
	// the repository actually intends to ship.
	repoValid := true
	if discovery != nil {
		repoResult, err := projects.ValidateRepo(*repoRoot)
		if err != nil {
			log.Fatalf("repository validation failed: %v", err)
		}
		repoOutput, err := projects.FormatRepoResult(repoResult, *outputFormat)
		if err != nil {
			log.Fatalf("failed to format repository results: %v", err)
		}
		fmt.Print(repoOutput)
		fmt.Println()
		repoValid = repoResult.Valid
	}

	projectResults, err := validator.ValidateAll(*configFile)
	if err != nil {
		log.Fatalf("validation failed: %v", err)
	}

	var excludedHandles map[string]bool
	if *baseMaintainersFile != "" {
		handles, err := validator.ExtractHandlesFrom(*baseMaintainersFile)
		if err != nil {
			log.Fatalf("failed to extract handles from base maintainers path: %v", err)
		}
		excludedHandles = handles
	}

	var maintainerResults []projects.MaintainerValidationResult
	maintainersEnabled := *maintainersFile != ""
	if maintainersEnabled {
		results, err := validator.ValidateMaintainersFileWithExclusion(*maintainersFile, *verifyMaintainers, excludedHandles)
		if err != nil {
			log.Fatalf("maintainers validation failed: %v", err)
		}
		maintainerResults = results
	}

	if discovery != nil && !skipMaintainers {
		for _, path := range discovery.MaintainersPaths() {
			results, err := validator.ValidateMaintainersFileWithExclusion(path, *verifyMaintainers, excludedHandles)
			maintainersEnabled = true
			if err != nil {
				// Record the failure as an invalid result instead of aborting,
				// so sibling projects that are fine still get reported. It
				// must not be dropped either: an unreadable or empty roster
				// has to show up in the maintainers report and fail the run.
				maintainerResults = append(maintainerResults, projects.MaintainerValidationResult{
					ProjectID: filepath.ToSlash(path),
					Valid:     false,
					Errors:    []string{err.Error()},
				})
				continue
			}
			maintainerResults = append(maintainerResults, results...)
		}
	}

	output, err := validator.FormatResults(projectResults, *outputFormat)
	if err != nil {
		log.Fatalf("failed to format project results: %v", err)
	}
	fmt.Print(output)
	if maintainersEnabled {
		fmt.Println()
		maintainersOutput, err := validator.FormatMaintainersResults(maintainerResults, *outputFormat)
		if err != nil {
			log.Fatalf("failed to format maintainer results: %v", err)
		}
		fmt.Print(maintainersOutput)
	}

	// Check if any validation failed
	hasErrors := !repoValid
	for _, result := range projectResults {
		if !result.Valid {
			hasErrors = true
			break
		}
	}
	if !hasErrors && maintainersEnabled {
		for _, result := range maintainerResults {
			if !result.Valid {
				hasErrors = true
				break
			}
		}
	}

	if *checkLinks {
		var maintainerFiles []string
		if discovery != nil {
			if !skipMaintainers {
				maintainerFiles = discovery.MaintainersPaths()
			}
		} else if *maintainersFile != "" {
			maintainerFiles = []string{*maintainersFile}
		}
		broken, err := runLinkCheck(*repoRoot, *configFile, maintainerFiles, linkOptions{
			base:   *linksBase,
			report: *linksReport,
			token:  githubToken(*envFile),
		}, *outputFormat)
		if err != nil {
			log.Fatalf("link check failed: %v", err)
		}
		if broken {
			hasErrors = true
		}
	}

	if hasErrors {
		os.Exit(1)
	}
}

// linkOptions configures a link check: base names a baseline whose links are
// skipped, report a markdown file for broken links, and token the GitHub
// token (empty for unauthenticated requests).
type linkOptions struct {
	base, report, token string
}

// newLinkChecker builds the link checker; tests replace it to avoid the
// network.
var newLinkChecker = projects.NewLinkChecker

// ghAuthToken returns the token of the gh CLI's active login, or "" when gh
// is not installed or not logged in. Tests replace it.
var ghAuthToken = func() string {
	ctx, cancel := context.WithTimeout(context.Background(), projects.DefaultHTTPTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "gh", "auth", "token").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// githubToken resolves the token used for link checks: GITHUB_TOKEN, then
// GH_TOKEN, from the environment or from envFile (real environment variables
// win), then the gh CLI's login. Without one, GitHub's unauthenticated rate
// limit turns most handle checks into warnings, so the run says so.
func githubToken(envFile string) string {
	if envFile != "" {
		if _, err := projects.LoadDotEnv(envFile); err != nil {
			fmt.Fprintf(os.Stderr, "link check: could not read %s: %v\n", envFile, err)
		}
	}
	for _, key := range []string{"GITHUB_TOKEN", "GH_TOKEN"} {
		if token := strings.TrimSpace(os.Getenv(key)); token != "" {
			return token
		}
	}
	if token := ghAuthToken(); token != "" {
		return token
	}
	sources := "GITHUB_TOKEN, GH_TOKEN"
	if envFile != "" {
		sources += ", " + envFile
	}
	fmt.Fprintf(os.Stderr, "link check: no GitHub token found (%s or 'gh auth login'); unauthenticated requests "+
		"are rate-limited and rate-limited links are only reported as warnings\n", sources)
	return ""
}

// runLinkCheck checks the links of the files being validated and reports
// whether any are broken. Placeholders and unconfirmable links are printed as
// warnings but never fail the run.
func runLinkCheck(root, projectList string, maintainerFiles []string, opts linkOptions, format string) (bool, error) {
	projectFiles, err := projects.LocalProjectFiles(projectList)
	if err != nil {
		return false, err
	}
	var present []string
	for _, f := range projectFiles {
		if _, err := os.Stat(f); err == nil {
			present = append(present, f)
		}
	}
	links := projects.CollectLinksFromFiles(root, present, maintainerFiles)

	if opts.base != "" {
		baseLinks, err := projects.CollectRepoLinks(opts.base)
		if err != nil {
			// A base that cannot be read (for example, the pull request
			// creates the repository's first project.yaml) has no links to
			// subtract, so everything is new.
			fmt.Fprintf(os.Stderr, "link check: base %s unreadable (%v); checking all links\n", opts.base, err)
		} else {
			links = projects.ExcludeKnownLinks(links, baseLinks)
		}
	}

	return checkAndReportLinks(links, opts, format)
}

// checkAndReportLinks checks links, prints the summary (and GitHub Actions
// annotations), optionally writes the markdown report, and reports whether
// any link is broken.
func checkAndReportLinks(links []projects.Link, opts linkOptions, format string) (bool, error) {
	results := newLinkChecker(opts.token).Check(context.Background(), links)

	// Structured output formats must stay parseable, so the human-readable
	// link summary goes to stderr for them.
	out := os.Stdout
	if format != "text" {
		out = os.Stderr
	}
	fmt.Fprintln(out)
	fmt.Fprint(out, projects.FormatLinkResultsText(results))
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		fmt.Print(projects.FormatLinkAnnotations(results))
	}

	broken, _ := projects.LinkSummary(results)
	if opts.report != "" {
		// The report is written even when it is empty, so a caller can tell
		// "no broken links" (empty file) from "the check never ran" (no file).
		var body []byte
		if broken > 0 {
			body = []byte(projects.FormatLinkReportMarkdown(results))
		}
		if err := os.WriteFile(opts.report, body, 0o644); err != nil {
			return broken > 0, err
		}
	}
	return broken > 0, nil
}

// runCommittees validates a committees.yaml file and returns the process exit
// code. Expired terms are warnings, so the result never depends on the date
// alone; only schema errors and unknown handles or URLs fail. links is nil
// when link checking is disabled.
func runCommittees(path, format string, links *linkOptions) int {
	cfg, err := projects.LoadCommitteesFromFile(path)
	if err != nil {
		log.Printf("committees validation failed: %v", err)
		return 1
	}
	res := projects.ValidateCommittees(cfg, time.Now())
	res.File = path
	out, err := projects.FormatCommitteesResult(res, format)
	if err != nil {
		log.Printf("failed to format committees results: %v", err)
		return 1
	}
	fmt.Print(out)

	code := 0
	if !res.Valid {
		code = 1
	}
	if links == nil {
		return code
	}

	toCheck := projects.CollectCommitteeLinks(cfg, filepath.ToSlash(path))
	if links.base != "" {
		basePath := links.base
		if info, err := os.Stat(basePath); err == nil && info.IsDir() {
			basePath = filepath.Join(basePath, projects.CommitteesFileName)
		}
		if baseCfg, err := projects.LoadCommitteesFromFile(basePath); err == nil {
			toCheck = projects.ExcludeKnownLinks(toCheck, projects.CollectCommitteeLinks(baseCfg, basePath))
		} else {
			fmt.Fprintf(os.Stderr, "link check: base %s unreadable (%v); checking all links\n", basePath, err)
		}
	}
	broken, err := checkAndReportLinks(toCheck, *links, format)
	if err != nil {
		log.Printf("link check failed: %v", err)
		return 1
	}
	if broken {
		code = 1
	}
	return code
}

// writeProjectList materializes a discovery result as a temporary project list
// file, so repo-root mode reuses the same fetch, cache and diff pipeline as an
// explicit -config run.
func writeProjectList(d *projects.Discovery) (string, func(), error) {
	f, err := os.CreateTemp("", "discovered-projectlist-*.yaml")
	if err != nil {
		return "", func() {}, err
	}
	cleanup := func() { os.Remove(f.Name()) }

	var b strings.Builder
	b.WriteString("projects:\n")
	for _, p := range d.Projects {
		abs, err := filepath.Abs(p.ProjectPath)
		if err != nil {
			f.Close()
			cleanup()
			return "", func() {}, err
		}

		// A declared project whose file is absent is already reported by
		// ValidateRepo, which names the exact expected path. Listing it here
		// too would restate the same problem as an opaque fetch failure.
		if _, err := os.Stat(abs); err != nil {
			continue
		}

		id := p.ID
		if id == "" {
			// Single-project repos have no declared id; the slug is not known
			// until the file is parsed, so fall back to the directory name.
			id = filepath.Base(filepath.Dir(abs))
		}

		b.WriteString(fmt.Sprintf("  - url: %q\n", "file://"+filepath.ToSlash(abs)))
		b.WriteString(fmt.Sprintf("    id: %q\n", id))
	}

	if _, err := f.WriteString(b.String()); err != nil {
		f.Close()
		cleanup()
		return "", func() {}, err
	}
	if err := f.Close(); err != nil {
		cleanup()
		return "", func() {}, err
	}

	return f.Name(), cleanup, nil
}
