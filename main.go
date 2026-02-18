package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Output is the top-level JSON structure emitted by daily-digest.
type Output struct {
	GeneratedAt string `json:"generatedAt"`
	Period      Period `json:"period"`
	GitHub      GitHub `json:"github"`
	Summary     Summary `json:"summary"`
	Error       string `json:"error,omitempty"`
}

// Period describes the time window for the digest.
type Period struct {
	Hours int    `json:"hours"`
	Since string `json:"since"`
}

// GitHub contains all GitHub-derived data.
type GitHub struct {
	PRsMerged   []PR    `json:"prsMerged"`
	PRsOpened   []PR    `json:"prsOpened"`
	IssuesClosed []Issue `json:"issuesClosed"`
	IssuesOpened []Issue `json:"issuesOpened"`
	Commits     Commits `json:"commits"`
}

// PR represents a pull request.
type PR struct {
	Repo   string `json:"repo"`
	Number int    `json:"number"`
	Title  string `json:"title"`
	URL    string `json:"url"`
	Author string `json:"author,omitempty"`
}

// Issue represents a GitHub issue.
type Issue struct {
	Repo   string `json:"repo"`
	Number int    `json:"number"`
	Title  string `json:"title"`
	URL    string `json:"url"`
	Author string `json:"author,omitempty"`
}

// Commits contains commit statistics.
type Commits struct {
	Total  int            `json:"total"`
	ByRepo map[string]int `json:"byRepo"`
}

// Summary contains aggregate statistics.
type Summary struct {
	TotalPRsMerged   int      `json:"totalPRsMerged"`
	TotalIssuesClosed int     `json:"totalIssuesClosed"`
	TotalCommits     int      `json:"totalCommits"`
	ActiveRepos      []string `json:"activeRepos"`
}

// ghSearchPRResult is the JSON structure returned by gh search prs.
type ghSearchPRResult struct {
	URL          string    `json:"url"`
	Number       int       `json:"number"`
	Title        string    `json:"title"`
	Repository   repoInfo  `json:"repository"`
	Author       author    `json:"author"`
	MergedAt     time.Time `json:"mergedAt"`
	CreatedAt    time.Time `json:"createdAt"`
	State        string    `json:"state"`
}

// ghSearchIssueResult is the JSON structure returned by gh search issues.
type ghSearchIssueResult struct {
	URL          string   `json:"url"`
	Number       int      `json:"number"`
	Title        string   `json:"title"`
	Repository   repoInfo `json:"repository"`
	Author       author   `json:"author"`
	ClosedAt     *time.Time `json:"closedAt"`
	CreatedAt    time.Time `json:"createdAt"`
	State        string    `json:"state"`
}

type repoInfo struct {
	NameWithOwner string `json:"nameWithOwner"`
}

type author struct {
	Login string `json:"login"`
}

func main() {
	org := flag.String("org", "", "GitHub organization to query (required)")
	hours := flag.Int("hours", 24, "Time window in hours")
	flag.Parse()

	if *org == "" {
		emitError("org flag is required")
		os.Exit(1)
	}

	since := time.Now().UTC().Add(-time.Duration(*hours) * time.Hour)
	out := Output{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Period: Period{
			Hours: *hours,
			Since: since.Format(time.RFC3339),
		},
		GitHub: GitHub{
			PRsMerged:    []PR{},
			PRsOpened:    []PR{},
			IssuesClosed: []Issue{},
			IssuesOpened: []Issue{},
			Commits: Commits{
				Total:  0,
				ByRepo: make(map[string]int),
			},
		},
	}

	// Gather GitHub data
	// Each function handles its own errors and returns empty results on failure
	prsMerged, err := fetchMergedPRs(*org, since)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[daily-digest] warning: failed to fetch merged PRs: %v\n", err)
		prsMerged = []PR{} // Ensure non-nil slice for JSON output
	}
	out.GitHub.PRsMerged = prsMerged

	prsOpened, err := fetchOpenedPRs(*org, since)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[daily-digest] warning: failed to fetch opened PRs: %v\n", err)
		prsOpened = []PR{} // Ensure non-nil slice for JSON output
	}
	out.GitHub.PRsOpened = prsOpened

	issuesClosed, err := fetchClosedIssues(*org, since)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[daily-digest] warning: failed to fetch closed issues: %v\n", err)
		issuesClosed = []Issue{} // Ensure non-nil slice for JSON output
	}
	out.GitHub.IssuesClosed = issuesClosed

	issuesOpened, err := fetchOpenedIssues(*org, since)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[daily-digest] warning: failed to fetch opened issues: %v\n", err)
		issuesOpened = []Issue{} // Ensure non-nil slice for JSON output
	}
	out.GitHub.IssuesOpened = issuesOpened

	commits, err := fetchCommits(*org, since)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[daily-digest] warning: failed to fetch commits: %v\n", err)
		commits = Commits{Total: 0, ByRepo: make(map[string]int)}
	}
	out.GitHub.Commits = commits

	// Compute summary
	out.Summary = computeSummary(out.GitHub)

	emitJSON(out)
}

func emitError(msg string) {
	emitJSON(Output{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Error:       msg,
	})
}

func emitJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func fetchMergedPRs(org string, since time.Time) ([]PR, error) {
	// Use gh search prs with merged:>=date filter
	sinceStr := since.Format("2006-01-02")
	args := []string{
		"search", "prs",
		"--org", org,
		"--merged", ">=" + sinceStr,
		"--sort", "updated",
		"--order", "desc",
		"--limit", "100",
		"--json", "url,number,title,repository,author,mergedAt",
	}

	stdout, err := runCmd("gh", args...)
	if err != nil {
		return nil, err
	}

	var results []ghSearchPRResult
	if err := json.Unmarshal(stdout, &results); err != nil {
		return nil, fmt.Errorf("parse gh search json: %w", err)
	}

	prs := make([]PR, 0, len(results))
	for _, r := range results {
		// Double-check mergedAt is within window (gh CLI filtering should handle this)
		if !r.MergedAt.IsZero() && r.MergedAt.Before(since) {
			continue
		}
		prs = append(prs, PR{
			Repo:   r.Repository.NameWithOwner,
			Number: r.Number,
			Title:  r.Title,
			URL:    r.URL,
			Author: r.Author.Login,
		})
	}
	return prs, nil
}

func fetchOpenedPRs(org string, since time.Time) ([]PR, error) {
	sinceStr := since.Format("2006-01-02")
	args := []string{
		"search", "prs",
		"--org", org,
		"--state", "open",
		"--created", ">=" + sinceStr,
		"--sort", "updated",
		"--order", "desc",
		"--limit", "100",
		"--json", "url,number,title,repository,author,createdAt",
	}

	stdout, err := runCmd("gh", args...)
	if err != nil {
		return nil, err
	}

	var results []ghSearchPRResult
	if err := json.Unmarshal(stdout, &results); err != nil {
		return nil, fmt.Errorf("parse gh search json: %w", err)
	}

	prs := make([]PR, 0, len(results))
	for _, r := range results {
		if !r.CreatedAt.IsZero() && r.CreatedAt.Before(since) {
			continue
		}
		prs = append(prs, PR{
			Repo:   r.Repository.NameWithOwner,
			Number: r.Number,
			Title:  r.Title,
			URL:    r.URL,
			Author: r.Author.Login,
		})
	}
	return prs, nil
}

func fetchClosedIssues(org string, since time.Time) ([]Issue, error) {
	sinceStr := since.Format("2006-01-02")
	args := []string{
		"search", "issues",
		"--org", org,
		"--state", "closed",
		"--closed", ">=" + sinceStr,
		"--sort", "updated",
		"--order", "desc",
		"--limit", "100",
		"--json", "url,number,title,repository,author,closedAt",
	}

	stdout, err := runCmd("gh", args...)
	if err != nil {
		return nil, err
	}

	var results []ghSearchIssueResult
	if err := json.Unmarshal(stdout, &results); err != nil {
		return nil, fmt.Errorf("parse gh search json: %w", err)
	}

	issues := make([]Issue, 0, len(results))
	for _, r := range results {
		if r.ClosedAt != nil && r.ClosedAt.Before(since) {
			continue
		}
		issues = append(issues, Issue{
			Repo:   r.Repository.NameWithOwner,
			Number: r.Number,
			Title:  r.Title,
			URL:    r.URL,
			Author: r.Author.Login,
		})
	}
	return issues, nil
}

func fetchOpenedIssues(org string, since time.Time) ([]Issue, error) {
	sinceStr := since.Format("2006-01-02")
	args := []string{
		"search", "issues",
		"--org", org,
		"--state", "open",
		"--created", ">=" + sinceStr,
		"--sort", "updated",
		"--order", "desc",
		"--limit", "100",
		"--json", "url,number,title,repository,author,createdAt",
	}

	stdout, err := runCmd("gh", args...)
	if err != nil {
		return nil, err
	}

	var results []ghSearchIssueResult
	if err := json.Unmarshal(stdout, &results); err != nil {
		return nil, fmt.Errorf("parse gh search json: %w", err)
	}

	issues := make([]Issue, 0, len(results))
	for _, r := range results {
		if !r.CreatedAt.IsZero() && r.CreatedAt.Before(since) {
			continue
		}
		issues = append(issues, Issue{
			Repo:   r.Repository.NameWithOwner,
			Number: r.Number,
			Title:  r.Title,
			URL:    r.URL,
			Author: r.Author.Login,
		})
	}
	return issues, nil
}

// commitResult represents the JSON output from gh api for commits.
type commitResult struct {
	Sha    string `json:"sha"`
	Commit struct {
		Author struct {
			Date string `json:"date"`
		} `json:"author"`
	} `json:"commit"`
}

func fetchCommits(org string, since time.Time) (Commits, error) {
	// Get list of repos in the org, then fetch commits for each
	repos, err := fetchOrgRepos(org)
	if err != nil {
		return Commits{}, err
	}

	commits := Commits{
		Total:  0,
		ByRepo: make(map[string]int),
	}

	sinceStr := since.Format(time.RFC3339)

	for _, repo := range repos {
		count, err := fetchRepoCommitCount(org, repo, sinceStr)
		if err != nil {
			// Log warning but continue with other repos
			fmt.Fprintf(os.Stderr, "[daily-digest] warning: failed to fetch commits for %s: %v\n", repo, err)
			continue
		}
		if count > 0 {
			commits.Total += count
			commits.ByRepo[repo] = count
		}
	}

	return commits, nil
}

// repoListResult represents a repo from gh repo list.
type repoListResult struct {
	Name          string `json:"name"`
	NameWithOwner string `json:"nameWithOwner"`
}

func fetchOrgRepos(org string) ([]string, error) {
	args := []string{
		"repo", "list", org,
		"--limit", "100",
		"--json", "name",
		"--no-archived",
	}

	stdout, err := runCmd("gh", args...)
	if err != nil {
		return nil, err
	}

	var results []repoListResult
	if err := json.Unmarshal(stdout, &results); err != nil {
		return nil, fmt.Errorf("parse gh repo list json: %w", err)
	}

	repos := make([]string, 0, len(results))
	for _, r := range results {
		repos = append(repos, r.Name)
	}
	return repos, nil
}

func fetchRepoCommitCount(org, repo, sinceRFC3339 string) (int, error) {
	// Use gh api to list commits since the given time
	args := []string{
		"api",
		fmt.Sprintf("repos/%s/%s/commits", org, repo),
		"-f", fmt.Sprintf("since=%s", sinceRFC3339),
		"-f", "per_page=100",
	}

	stdout, err := runCmd("gh", args...)
	if err != nil {
		return 0, err
	}

	var results []commitResult
	if err := json.Unmarshal(stdout, &results); err != nil {
		return 0, fmt.Errorf("parse commits json: %w", err)
	}

	return len(results), nil
}

func runCmd(bin string, args ...string) ([]byte, error) {
	cmd := exec.Command(bin, args...)
	cmd.Env = os.Environ()
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("%s %s: %s", bin, strings.Join(args, " "), msg)
	}
	return []byte(stdout.String()), nil
}

func computeSummary(gh GitHub) Summary {
	activeRepos := make(map[string]bool)
	for _, pr := range gh.PRsMerged {
		activeRepos[pr.Repo] = true
	}
	for _, pr := range gh.PRsOpened {
		activeRepos[pr.Repo] = true
	}
	for _, issue := range gh.IssuesClosed {
		activeRepos[issue.Repo] = true
	}
	for _, issue := range gh.IssuesOpened {
		activeRepos[issue.Repo] = true
	}
	for repo := range gh.Commits.ByRepo {
		activeRepos[repo] = true
	}

	repos := make([]string, 0, len(activeRepos))
	for repo := range activeRepos {
		repos = append(repos, repo)
	}

	return Summary{
		TotalPRsMerged:    len(gh.PRsMerged),
		TotalIssuesClosed: len(gh.IssuesClosed),
		TotalCommits:      gh.Commits.Total,
		ActiveRepos:       repos,
	}
}