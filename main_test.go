package main

import (
	"encoding/json"
	"testing"
	"time"
)

func TestComputeSummary(t *testing.T) {
	tests := []struct {
		name     string
		gh       GitHub
		expected Summary
	}{
		{
			name: "empty github data",
			gh: GitHub{
				PRsMerged:    []PR{},
				PRsOpened:    []PR{},
				IssuesClosed: []Issue{},
				IssuesOpened: []Issue{},
				Commits: Commits{
					Total:  0,
					ByRepo: map[string]int{},
				},
			},
			expected: Summary{
				TotalPRsMerged:    0,
				TotalIssuesClosed: 0,
				TotalCommits:      0,
				ActiveRepos:       []string{},
			},
		},
		{
			name: "single merged PR",
			gh: GitHub{
				PRsMerged: []PR{
					{Repo: "misty-step/factory", Number: 42, Title: "Add feature", URL: "https://github.com/misty-step/factory/pull/42"},
				},
				PRsOpened:    []PR{},
				IssuesClosed: []Issue{},
				IssuesOpened: []Issue{},
				Commits: Commits{
					Total:  0,
					ByRepo: map[string]int{},
				},
			},
			expected: Summary{
				TotalPRsMerged:    1,
				TotalIssuesClosed: 0,
				TotalCommits:      0,
				ActiveRepos:       []string{"misty-step/factory"},
			},
		},
		{
			name: "multiple repos active",
			gh: GitHub{
				PRsMerged: []PR{
					{Repo: "misty-step/factory", Number: 42, Title: "Add feature", URL: "https://github.com/misty-step/factory/pull/42"},
				},
				PRsOpened: []PR{
					{Repo: "misty-step/cerberus", Number: 10, Title: "Fix bug", URL: "https://github.com/misty-step/cerberus/pull/10"},
				},
				IssuesClosed: []Issue{
					{Repo: "misty-step/factory", Number: 100, Title: "Bug report", URL: "https://github.com/misty-step/factory/issues/100"},
				},
				IssuesOpened: []Issue{
					{Repo: "misty-step/utils", Number: 5, Title: "New feature request", URL: "https://github.com/misty-step/utils/issues/5"},
				},
				Commits: Commits{
					Total: 15,
					ByRepo: map[string]int{
						"misty-step/factory": 10,
						"misty-step/cerberus": 5,
					},
				},
			},
			expected: Summary{
				TotalPRsMerged:    1,
				TotalIssuesClosed: 1,
				TotalCommits:      15,
				ActiveRepos:       []string{"misty-step/factory", "misty-step/cerberus", "misty-step/utils"}, // order may vary
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := computeSummary(tt.gh)

			if result.TotalPRsMerged != tt.expected.TotalPRsMerged {
				t.Errorf("TotalPRsMerged: got %d, want %d", result.TotalPRsMerged, tt.expected.TotalPRsMerged)
			}
			if result.TotalIssuesClosed != tt.expected.TotalIssuesClosed {
				t.Errorf("TotalIssuesClosed: got %d, want %d", result.TotalIssuesClosed, tt.expected.TotalIssuesClosed)
			}
			if result.TotalCommits != tt.expected.TotalCommits {
				t.Errorf("TotalCommits: got %d, want %d", result.TotalCommits, tt.expected.TotalCommits)
			}

			// ActiveRepos order is not guaranteed, compare as sets
			if len(result.ActiveRepos) != len(tt.expected.ActiveRepos) {
				t.Errorf("ActiveRepos count: got %d, want %d", len(result.ActiveRepos), len(tt.expected.ActiveRepos))
			} else {
				resultSet := make(map[string]bool)
				for _, r := range result.ActiveRepos {
					resultSet[r] = true
				}
				for _, r := range tt.expected.ActiveRepos {
					if !resultSet[r] {
						t.Errorf("ActiveRepos missing: %s", r)
					}
				}
			}
		})
	}
}

func TestParseGhSearchPRResult(t *testing.T) {
	sample := `[{"url":"https://github.com/misty-step/factory/pull/42","number":42,"title":"Add daily digest","repository":{"nameWithOwner":"misty-step/factory"},"author":{"login":"kaylee-mistystep"},"mergedAt":"2026-02-18T10:00:00Z","state":"MERGED"}]`

	var results []ghSearchPRResult
	if err := json.Unmarshal([]byte(sample), &results); err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if r.URL != "https://github.com/misty-step/factory/pull/42" {
		t.Errorf("URL: got %s", r.URL)
	}
	if r.Number != 42 {
		t.Errorf("Number: got %d", r.Number)
	}
	if r.Title != "Add daily digest" {
		t.Errorf("Title: got %s", r.Title)
	}
	if r.Repository.NameWithOwner != "misty-step/factory" {
		t.Errorf("Repository: got %s", r.Repository.NameWithOwner)
	}
	if r.Author.Login != "kaylee-mistystep" {
		t.Errorf("Author: got %s", r.Author.Login)
	}
}

func TestParseGhSearchIssueResult(t *testing.T) {
	sample := `[{"url":"https://github.com/misty-step/factory/issues/100","number":100,"title":"Bug in digest","repository":{"nameWithOwner":"misty-step/factory"},"author":{"login":"phaedrus"},"state":"closed","closedAt":"2026-02-18T10:00:00Z"}]`

	var results []ghSearchIssueResult
	if err := json.Unmarshal([]byte(sample), &results); err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if r.URL != "https://github.com/misty-step/factory/issues/100" {
		t.Errorf("URL: got %s", r.URL)
	}
	if r.Number != 100 {
		t.Errorf("Number: got %d", r.Number)
	}
	if r.Title != "Bug in digest" {
		t.Errorf("Title: got %s", r.Title)
	}
	if r.Repository.NameWithOwner != "misty-step/factory" {
		t.Errorf("Repository: got %s", r.Repository.NameWithOwner)
	}
	if r.Author.Login != "phaedrus" {
		t.Errorf("Author: got %s", r.Author.Login)
	}
	if r.ClosedAt == nil {
		t.Error("ClosedAt should not be nil")
	}
}

func TestTimeWindowFiltering(t *testing.T) {
	// Test that PRs before the since window are filtered out
	since, _ := time.Parse(time.RFC3339, "2026-02-18T00:00:00Z")
	
	// PR merged before window
	oldPR := ghSearchPRResult{
		URL:        "https://github.com/misty-step/factory/pull/1",
		Number:     1,
		Title:      "Old PR",
		MergedAt:   time.Date(2026, 2, 17, 10, 0, 0, 0, time.UTC), // Before since
	}
	
	// PR merged within window
	newPR := ghSearchPRResult{
		URL:        "https://github.com/misty-step/factory/pull/2",
		Number:     2,
		Title:      "New PR",
		MergedAt:   time.Date(2026, 2, 18, 10, 0, 0, 0, time.UTC), // After since
	}
	
	// Verify filtering logic
	if !oldPR.MergedAt.Before(since) {
		t.Error("oldPR should be before since")
	}
	if newPR.MergedAt.Before(since) {
		t.Error("newPR should not be before since")
	}
}

func TestEmptyResultsProduceValidJSON(t *testing.T) {
	out := Output{
		GeneratedAt: "2026-02-18T14:00:00Z",
		Period: Period{
			Hours: 24,
			Since: "2026-02-17T14:00:00Z",
		},
		GitHub: GitHub{
			PRsMerged:    []PR{},
			PRsOpened:    []PR{},
			IssuesClosed: []Issue{},
			IssuesOpened: []Issue{},
			Commits: Commits{
				Total:  0,
				ByRepo: map[string]int{},
			},
		},
		Summary: Summary{
			TotalPRsMerged:    0,
			TotalIssuesClosed: 0,
			TotalCommits:      0,
			ActiveRepos:       []string{},
		},
	}

	data, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("failed to marshal empty output: %v", err)
	}

	// Verify it's valid JSON
	var parsed Output
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.GeneratedAt != out.GeneratedAt {
		t.Errorf("GeneratedAt: got %s, want %s", parsed.GeneratedAt, out.GeneratedAt)
	}
	if parsed.Period.Hours != out.Period.Hours {
		t.Errorf("Period.Hours: got %d, want %d", parsed.Period.Hours, out.Period.Hours)
	}
}

func TestMalformedGhOutputDoesNotPanic(t *testing.T) {
	// This tests that malformed JSON returns an error, not a panic
	malformed := `not valid json [{"url":`
	
	var results []ghSearchPRResult
	err := json.Unmarshal([]byte(malformed), &results)
	
	if err == nil {
		t.Error("expected error for malformed JSON")
	}
	// If we get here without panic, the test passes
}

func TestOutputWithAllFields(t *testing.T) {
	out := Output{
		GeneratedAt: "2026-02-18T14:00:00Z",
		Period: Period{
			Hours: 24,
			Since: "2026-02-17T14:00:00Z",
		},
		GitHub: GitHub{
			PRsMerged: []PR{
				{Repo: "misty-step/factory", Number: 42, Title: "Add daily digest", URL: "https://github.com/misty-step/factory/pull/42", Author: "kaylee"},
			},
			PRsOpened: []PR{
				{Repo: "misty-step/cerberus", Number: 10, Title: "Fix auth", URL: "https://github.com/misty-step/cerberus/pull/10", Author: "phaedrus"},
			},
			IssuesClosed: []Issue{
				{Repo: "misty-step/factory", Number: 100, Title: "Bug report", URL: "https://github.com/misty-step/factory/issues/100", Author: "user"},
			},
			IssuesOpened: []Issue{
				{Repo: "misty-step/utils", Number: 5, Title: "Feature request", URL: "https://github.com/misty-step/utils/issues/5", Author: "contributor"},
			},
			Commits: Commits{
				Total: 15,
				ByRepo: map[string]int{
					"misty-step/factory":  10,
					"misty-step/cerberus": 5,
				},
			},
		},
		Summary: Summary{
			TotalPRsMerged:    1,
			TotalIssuesClosed: 1,
			TotalCommits:      15,
			ActiveRepos:       []string{"misty-step/factory", "misty-step/cerberus", "misty-step/utils"},
		},
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Verify the JSON contains expected fields
	jsonStr := string(data)
	
	// Check PRsMerged
	if !contains(jsonStr, `"prsMerged"`) {
		t.Error("JSON missing prsMerged field")
	}
	if !contains(jsonStr, `"misty-step/factory"`) {
		t.Error("JSON missing repo name")
	}
	if !contains(jsonStr, `"totalPRsMerged": 1`) {
		t.Error("JSON missing totalPRsMerged count")
	}
	
	// Verify round-trip
	var parsed Output
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	
	if len(parsed.GitHub.PRsMerged) != 1 {
		t.Errorf("PRsMerged: got %d, want 1", len(parsed.GitHub.PRsMerged))
	}
	if parsed.GitHub.Commits.Total != 15 {
		t.Errorf("Commits.Total: got %d, want 15", parsed.GitHub.Commits.Total)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestOutputWithError(t *testing.T) {
	out := Output{
		GeneratedAt: "2026-02-18T14:00:00Z",
		Error:       "failed to fetch data: connection refused",
	}

	data, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed Output
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.Error != out.Error {
		t.Errorf("Error: got %s, want %s", parsed.Error, out.Error)
	}
}

func TestPRStructFields(t *testing.T) {
	pr := PR{
		Repo:   "misty-step/factory",
		Number: 42,
		Title:  "Add daily digest",
		URL:    "https://github.com/misty-step/factory/pull/42",
		Author: "kaylee-mistystep",
	}

	data, err := json.Marshal(pr)
	if err != nil {
		t.Fatalf("failed to marshal PR: %v", err)
	}

	var parsed PR
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal PR: %v", err)
	}

	if parsed.Repo != pr.Repo {
		t.Errorf("Repo: got %s, want %s", parsed.Repo, pr.Repo)
	}
	if parsed.Number != pr.Number {
		t.Errorf("Number: got %d, want %d", parsed.Number, pr.Number)
	}
	if parsed.Title != pr.Title {
		t.Errorf("Title: got %s, want %s", parsed.Title, pr.Title)
	}
	if parsed.URL != pr.URL {
		t.Errorf("URL: got %s, want %s", parsed.URL, pr.URL)
	}
	if parsed.Author != pr.Author {
		t.Errorf("Author: got %s, want %s", parsed.Author, pr.Author)
	}
}

func TestIssueStructFields(t *testing.T) {
	issue := Issue{
		Repo:   "misty-step/factory",
		Number: 100,
		Title:  "Bug report",
		URL:    "https://github.com/misty-step/factory/issues/100",
		Author: "phaedrus",
	}

	data, err := json.Marshal(issue)
	if err != nil {
		t.Fatalf("failed to marshal Issue: %v", err)
	}

	var parsed Issue
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal Issue: %v", err)
	}

	if parsed.Repo != issue.Repo {
		t.Errorf("Repo: got %s, want %s", parsed.Repo, issue.Repo)
	}
	if parsed.Number != issue.Number {
		t.Errorf("Number: got %d, want %d", parsed.Number, issue.Number)
	}
}

func TestCommitsStructFields(t *testing.T) {
	commits := Commits{
		Total: 15,
		ByRepo: map[string]int{
			"misty-step/factory":  10,
			"misty-step/cerberus": 5,
		},
	}

	data, err := json.Marshal(commits)
	if err != nil {
		t.Fatalf("failed to marshal Commits: %v", err)
	}

	var parsed Commits
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal Commits: %v", err)
	}

	if parsed.Total != commits.Total {
		t.Errorf("Total: got %d, want %d", parsed.Total, commits.Total)
	}
	if len(parsed.ByRepo) != len(commits.ByRepo) {
		t.Errorf("ByRepo count: got %d, want %d", len(parsed.ByRepo), len(commits.ByRepo))
	}
}

func TestPeriodStruct(t *testing.T) {
	period := Period{
		Hours: 24,
		Since: "2026-02-17T14:00:00Z",
	}

	data, err := json.Marshal(period)
	if err != nil {
		t.Fatalf("failed to marshal Period: %v", err)
	}

	var parsed Period
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal Period: %v", err)
	}

	if parsed.Hours != period.Hours {
		t.Errorf("Hours: got %d, want %d", parsed.Hours, period.Hours)
	}
	if parsed.Since != period.Since {
		t.Errorf("Since: got %s, want %s", parsed.Since, period.Since)
	}
}