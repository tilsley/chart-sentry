package domain

// PRContext holds the details of a pull request event.
type PRContext struct {
	Owner    string
	Repo     string
	PRNumber int
	BaseRef  string
	HeadRef  string
	HeadSHA  string
}

// Status represents the outcome of a diff operation.
type Status int

const (
	StatusSuccess Status = iota // No changes detected
	StatusChanges                // Changes detected
	StatusError                  // Error occurred during diff
)

// DiffResult represents the diff output for a single chart + environment pair.
type DiffResult struct {
	ChartName    string
	Environment  string
	BaseRef      string
	HeadRef      string
	Status       Status // Outcome of the diff operation
	HasChanges   bool   // Deprecated: use Status instead
	UnifiedDiff  string // Traditional line-based diff (go-difflib)
	SemanticDiff string // Semantic YAML diff (dyff) - may be empty if dyff unavailable
	Summary      string // Human-readable summary (or error message if Status == StatusError)
}

// PreferredDiff returns the semantic diff if available, otherwise the unified diff.
// This allows reporting adapters to prefer semantic diffs while falling back to unified.
func (r DiffResult) PreferredDiff() string {
	if r.SemanticDiff != "" {
		return r.SemanticDiff
	}
	return r.UnifiedDiff
}

// CountByStatus returns counts of results grouped by status.
func CountByStatus(results []DiffResult) (success, changes, errors int) {
	for _, r := range results {
		switch r.Status {
		case StatusSuccess:
			success++
		case StatusChanges:
			changes++
		case StatusError:
			errors++
		}
	}
	return
}
