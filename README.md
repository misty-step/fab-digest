# fab-digest

Daily operational digest generator for GitHub organizations. Collects PR, issue, and commit metrics within a configurable time window and outputs a structured JSON summary.

## What It Does

`fab-digest` queries the GitHub API (via the `gh` CLI) to gather operational metrics for a specified organization:

- **PRs Merged**: All pull requests merged within the time window
- **PRs Opened**: All pull requests created within the time window
- **Issues Closed**: All issues closed within the time window
- **Issues Opened**: All issues created within the time window
- **Commits**: Commit counts per repository within the time window
- **Summary**: Aggregate totals and list of active repositories

The tool is designed for daily automation (via OpenClaw cron) to produce a JSON digest of organizational activity.

## Installation

```bash
go install github.com/misty-step/fab-digest@latest
```

Alternatively, clone and build from source:

```bash
git clone https://github.com/misty-step/fab-digest.git
cd fab-digest
go install .
```

## Requirements

- **Go 1.25+**
- **GitHub CLI (`gh`)** - The tool uses `gh` for all GitHub API queries. Ensure `gh` is installed and authenticated:
  ```bash
  gh auth login
  ```

## Usage

### Basic Usage

```bash
fab-digest -org misty-step
```

This queries the `misty-step` organization for the last 24 hours.

### Custom Time Window

```bash
# Last 48 hours
fab-digest -org misty-step -hours 48

# Last 7 days (168 hours)
fab-digest -org misty-step -hours 168
```

### Command-Line Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-org` | string | (required) | GitHub organization to query |
| `-hours` | int | 24 | Time window in hours |

### Output Format

The tool outputs JSON to stdout. Example output:

```json
{
  "generatedAt": "2026-02-18T12:00:00Z",
  "period": {
    "hours": 24,
    "since": "2026-02-17T12:00:00Z"
  },
  "github": {
    "prsMerged": [
      {
        "repo": "misty-step/factory",
        "number": 42,
        "title": "feat: add new integration",
        "url": "https://github.com/misty-step/factory/pull/42",
        "author": "jdoe"
      }
    ],
    "prsOpened": [],
    "issuesClosed": [],
    "issuesOpened": [],
    "commits": {
      "total": 15,
      "byRepo": {
        "factory": 10,
        "fab-digest": 5
      }
    }
  },
  "summary": {
    "totalPRsMerged": 1,
    "totalIssuesClosed": 0,
    "totalCommits": 15,
    "activeRepos": ["factory", "fab-digest"]
  }
}
```

### Error Handling

If the `-org` flag is missing, the tool outputs an error JSON and exits with code 1:

```json
{
  "generatedAt": "2026-02-18T12:00:00Z",
  "error": "org flag is required"
}
```

Partial failures (e.g., one GitHub query fails) are logged to stderr but do not abort the entire operation—empty results are returned for failed queries.

## Configuration

`fab-digest` is configured entirely via command-line flags:

- `-org`: The GitHub organization to query (required)
- `-hours`: The time window in hours (optional, defaults to 24)

No configuration files or environment variables are required.

### GitHub Authentication

Ensure you are authenticated with GitHub:

```bash
gh auth status
gh auth login
```

The tool requires appropriate permissions to:
- Search PRs and issues in the organization
- List repositories in the organization
- Query commit history

## Integration

`fab-digest` is designed to run as part of the factory's daily operational cycle via OpenClaw cron.

### Example Cron Integration

```bash
# Daily digest at 9:00 AM
0 9 * * * fab-digest -org misty-step -hours 24 >> /var/log/fab-digest.json
```

The JSON output can be:
- Parsed by downstream automation
- Stored for historical tracking
- Displayed in dashboards

## Contributing

Contributions are welcome. Standard Go contribution workflow:

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Make changes and add tests
4. Run tests: `go test ./...`
5. Commit with conventional commits: `git commit -am 'feat: add new feature'`
6. Push and create a PR

### Running Tests

```bash
go test -v ./...
```

### Code Style

This project follows standard Go conventions. Run `go fmt` before committing:

```bash
go fmt ./...
```

## License

Internal use only.
