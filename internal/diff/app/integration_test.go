package app

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	dyffdiff "github.com/nathantilsley/chart-sentry/internal/diff/adapters/dyff_diff"
	envdiscovery "github.com/nathantilsley/chart-sentry/internal/diff/adapters/env_discovery"
	helmcli "github.com/nathantilsley/chart-sentry/internal/diff/adapters/helm_cli"
	linediff "github.com/nathantilsley/chart-sentry/internal/diff/adapters/line_diff"
	"github.com/nathantilsley/chart-sentry/internal/diff/domain"
)

var update = flag.Bool("update", false, "update golden files")

func TestIntegration_FullDiffFlow(t *testing.T) {
	if _, err := helmcli.New(); err != nil {
		t.Skipf("helm not on PATH, skipping integration test: %v", err)
	}

	renderer, err := helmcli.New()
	if err != nil {
		t.Fatalf("creating helm adapter: %v", err)
	}

	ctx := context.Background()
	testdataDir := filepath.Join("testdata")
	baseChartDir := filepath.Join(testdataDir, "base", "my-app")
	headChartDir := filepath.Join(testdataDir, "head", "my-app")
	goldenDir := filepath.Join(testdataDir, "golden")

	baseRef := "main"
	headRef := "feat/update-config"

	// Discover environments from head chart dir
	discovery := envdiscovery.New()
	envs, err := discovery.DiscoverEnvironments(ctx, headChartDir)
	if err != nil {
		t.Fatalf("discovering environments: %v", err)
	}

	// Init diff adapters
	semanticDiff := dyffdiff.New()
	unifiedDiff := linediff.New()

	var allResults []domain.DiffResult

	for _, env := range envs {
		t.Run(env.Name, func(t *testing.T) {
			baseManifest, err := renderer.Render(ctx, baseChartDir, env.ValueFiles)
			if err != nil {
				t.Fatalf("rendering base for %s: %v", env.Name, err)
			}

			headManifest, err := renderer.Render(ctx, headChartDir, env.ValueFiles)
			if err != nil {
				t.Fatalf("rendering head for %s: %v", env.Name, err)
			}

			baseName := domain.DiffLabel("my-app", env.Name, baseRef)
			headName := domain.DiffLabel("my-app", env.Name, headRef)

			// Compute both diffs
			semanticDiffOutput := semanticDiff.ComputeDiff(baseName, headName, baseManifest, headManifest)
			unifiedDiffOutput := unifiedDiff.ComputeDiff(baseName, headName, baseManifest, headManifest)

			var status domain.Status
			var summary string
			if semanticDiffOutput != "" || unifiedDiffOutput != "" {
				status = domain.StatusChanges
				summary = fmt.Sprintf("Changes detected in my-app for environment %s.", env.Name)
			} else {
				status = domain.StatusSuccess
				summary = "No changes detected."
			}

			result := domain.DiffResult{
				ChartName:    "my-app",
				Environment:  env.Name,
				BaseRef:      baseRef,
				HeadRef:      headRef,
				Status:       status,
				UnifiedDiff:  unifiedDiffOutput,
				SemanticDiff: semanticDiffOutput,
				Summary:      summary,
			}
			allResults = append(allResults, result)

			if status != domain.StatusChanges {
				t.Fatal("expected changes but got none")
			}
		})
	}

	// Generate grouped check run markdown (one per chart)
	checkRunMD := formatCheckRunMarkdown(allResults)
	goldenFile := filepath.Join(goldenDir, "check-run-my-app.md")
	compareOrUpdateGolden(t, goldenFile, checkRunMD)

	// Generate PR summary comment
	prComment := formatPRComment(allResults)
	goldenFile = filepath.Join(goldenDir, "pr-comment.md")
	compareOrUpdateGolden(t, goldenFile, prComment)
}

// formatCheckRunMarkdown produces the markdown that mirrors what GitHub displays
// for a Check Run created by chart-sentry — one check run per chart with
// collapsible environment sections.
func formatCheckRunMarkdown(results []domain.DiffResult) string {
	if len(results) == 0 {
		return ""
	}

	chartName := results[0].ChartName

	changed := 0
	unchanged := 0
	for _, r := range results {
		if r.Status == domain.StatusChanges {
			changed++
		} else {
			unchanged++
		}
	}

	conclusion := "success"
	if changed > 0 {
		conclusion = "neutral"
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "# chart-sentry: %s\n\n", chartName)
	fmt.Fprintf(&sb, "**Status:** completed\n")
	fmt.Fprintf(&sb, "**Conclusion:** %s\n\n", conclusion)
	fmt.Fprintf(&sb, "## Helm diff — %s\n\n", chartName)
	fmt.Fprintf(&sb, "### Summary\n")
	fmt.Fprintf(&sb, "Analyzed %d environment(s): %d changed, %d unchanged\n\n", len(results), changed, unchanged)
	fmt.Fprintf(&sb, "### Output\n")

	for i, r := range results {
		if i > 0 {
			sb.WriteString("\n")
		}

		status := "No Changes"
		if r.Status == domain.StatusChanges {
			status = "Changed"
		}

		fmt.Fprintf(&sb, "<details><summary>%s — %s</summary>\n\n", r.Environment, status)

		if r.UnifiedDiff == "" && r.SemanticDiff == "" {
			sb.WriteString("No changes detected.\n")
		} else {
			// Show semantic diff first (if available), then unified diff
			if r.SemanticDiff != "" {
				sb.WriteString("**Semantic Diff (dyff):**\n")
				fmt.Fprintf(&sb, "```diff\n%s\n```\n\n", r.SemanticDiff)
			}
			if r.UnifiedDiff != "" {
				sb.WriteString("**Unified Diff (line-based):**\n")
				fmt.Fprintf(&sb, "```diff\n%s\n```\n", r.UnifiedDiff)
			}
		}

		sb.WriteString("\n</details>\n")
	}

	return sb.String()
}

// formatPRComment produces a summary comment aggregating all environment diffs.
func formatPRComment(results []domain.DiffResult) string {
	var sb strings.Builder
	sb.WriteString("## Chart-Sentry Diff Report\n\n")

	// Table header
	sb.WriteString("| Chart | Environment | Status |\n")
	sb.WriteString("|-------|-------------|--------|\n")
	for _, r := range results {
		status := "No Changes"
		if r.Status == domain.StatusChanges {
			status = "Changed"
		}
		fmt.Fprintf(&sb, "| %s | %s | %s |\n", r.ChartName, r.Environment, status)
	}
	sb.WriteString("\n")

	// Detail sections - prefer semantic diff in PR comments
	for _, r := range results {
		fmt.Fprintf(&sb, "### %s/%s\n", r.ChartName, r.Environment)
		if r.Status != domain.StatusChanges {
			sb.WriteString("No changes detected.\n\n")
			continue
		}
		sb.WriteString("<details><summary>View diff</summary>\n\n")
		// Prefer semantic diff, fall back to unified diff
		diffToShow := r.SemanticDiff
		if diffToShow == "" {
			diffToShow = r.UnifiedDiff
		}
		fmt.Fprintf(&sb, "```diff\n%s\n```\n", diffToShow)
		sb.WriteString("</details>\n\n")
	}

	return sb.String()
}

// TestIntegration_NewChart tests the scenario where a chart is being added
// for the first time (exists in HEAD but not in BASE).
func TestIntegration_NewChart(t *testing.T) {
	if _, err := helmcli.New(); err != nil {
		t.Skipf("helm not on PATH, skipping integration test: %v", err)
	}

	renderer, err := helmcli.New()
	if err != nil {
		t.Fatalf("creating helm adapter: %v", err)
	}

	ctx := context.Background()
	testdataDir := filepath.Join("testdata")
	// Base chart does NOT exist - simulating new chart
	baseChartDir := filepath.Join(testdataDir, "base", "new-chart")
	headChartDir := filepath.Join(testdataDir, "head", "new-chart")
	goldenDir := filepath.Join(testdataDir, "golden")

	baseRef := "main"
	headRef := "feat/add-new-chart"

	// Check that base chart does NOT exist
	if _, err := os.Stat(baseChartDir); err == nil {
		t.Fatalf("base chart should not exist for this test, but it does at %s", baseChartDir)
	}

	// Discover environments from head chart dir
	discovery := envdiscovery.New()
	envs, err := discovery.DiscoverEnvironments(ctx, headChartDir)
	if err != nil {
		t.Fatalf("discovering environments: %v", err)
	}

	// Init diff adapters
	semanticDiff := dyffdiff.New()
	unifiedDiff := linediff.New()

	var allResults []domain.DiffResult

	for _, env := range envs {
		t.Run(env.Name, func(t *testing.T) {
			// For a new chart, base manifest should be empty
			var baseManifest []byte
			if _, err := os.Stat(baseChartDir); err == nil {
				baseManifest, err = renderer.Render(ctx, baseChartDir, env.ValueFiles)
				if err != nil {
					t.Fatalf("rendering base for %s: %v", env.Name, err)
				}
			}
			// else: baseManifest remains empty (nil/empty byte slice)

			headManifest, err := renderer.Render(ctx, headChartDir, env.ValueFiles)
			if err != nil {
				t.Fatalf("rendering head for %s: %v", env.Name, err)
			}

			baseName := domain.DiffLabel("new-chart", env.Name, baseRef)
			headName := domain.DiffLabel("new-chart", env.Name, headRef)

			// Compute both diffs
			semanticDiffOutput := semanticDiff.ComputeDiff(baseName, headName, baseManifest, headManifest)
			unifiedDiffOutput := unifiedDiff.ComputeDiff(baseName, headName, baseManifest, headManifest)

			var status domain.Status
			var summary string
			if semanticDiffOutput != "" || unifiedDiffOutput != "" {
				status = domain.StatusChanges
				summary = fmt.Sprintf("Changes detected in new-chart for environment %s.", env.Name)
			} else {
				status = domain.StatusSuccess
				summary = "No changes detected."
			}

			result := domain.DiffResult{
				ChartName:    "new-chart",
				Environment:  env.Name,
				BaseRef:      baseRef,
				HeadRef:      headRef,
				Status:       status,
				UnifiedDiff:  unifiedDiffOutput,
				SemanticDiff: semanticDiffOutput,
				Summary:      summary,
			}
			allResults = append(allResults, result)

			if status != domain.StatusChanges {
				t.Fatal("expected changes but got none (new chart should show all additions)")
			}
		})
	}

	// Generate grouped check run markdown
	checkRunMD := formatCheckRunMarkdown(allResults)
	goldenFile := filepath.Join(goldenDir, "check-run-new-chart.md")
	compareOrUpdateGolden(t, goldenFile, checkRunMD)
}

// compareOrUpdateGolden either updates the golden file or compares against it.
func compareOrUpdateGolden(t *testing.T, path, actual string) {
	t.Helper()

	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("creating golden dir: %v", err)
		}
		if err := os.WriteFile(path, []byte(actual), 0o644); err != nil {
			t.Fatalf("writing golden file %s: %v", path, err)
		}
		t.Logf("updated golden file: %s", path)
		return
	}

	expected, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading golden file %s (run with -update to create): %v", path, err)
	}

	if string(expected) != actual {
		t.Errorf("output does not match golden file %s\n\n--- expected ---\n%s\n--- actual ---\n%s",
			path, string(expected), actual)
	}
}
