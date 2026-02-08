package linediff

import (
	"strings"
	"testing"
)

func TestAdapter_ComputeDiff(t *testing.T) {
	tests := []struct {
		name     string
		baseName string
		headName string
		base     []byte
		head     []byte
		want     string // Empty if no diff expected
	}{
		{
			name:     "identical content returns empty diff",
			baseName: "test (main)",
			headName: "test (feature)",
			base:     []byte("line1\nline2\nline3\n"),
			head:     []byte("line1\nline2\nline3\n"),
			want:     "",
		},
		{
			name:     "different content returns unified diff",
			baseName: "test (main)",
			headName: "test (feature)",
			base:     []byte("line1\nline2\nline3\n"),
			head:     []byte("line1\nmodified\nline3\n"),
			want:     "--- test (main)\n+++ test (feature)\n@@ -1,4 +1,4 @@\n line1\n-line2\n+modified\n line3",
		},
		{
			name:     "added lines",
			baseName: "test (main)",
			headName: "test (feature)",
			base:     []byte("line1\nline2\n"),
			head:     []byte("line1\nline2\nline3\nline4\n"),
			want:     "--- test (main)\n+++ test (feature)\n@@ -1,3 +1,5 @@\n line1\n line2\n+line3\n+line4",
		},
		{
			name:     "removed lines",
			baseName: "test (main)",
			headName: "test (feature)",
			base:     []byte("line1\nline2\nline3\nline4\n"),
			head:     []byte("line1\nline2\n"),
			want:     "--- test (main)\n+++ test (feature)\n@@ -1,5 +1,3 @@\n line1\n line2\n-line3\n-line4",
		},
		{
			name:     "empty base shows all additions",
			baseName: "test (main)",
			headName: "test (feature)",
			base:     []byte(""),
			head:     []byte("new content\n"),
			want:     "--- test (main)\n+++ test (feature)\n@@ -1 +1,2 @@\n+new content",
		},
		{
			name:     "empty head shows all deletions",
			baseName: "test (main)",
			headName: "test (feature)",
			base:     []byte("old content\n"),
			head:     []byte(""),
			want:     "--- test (main)\n+++ test (feature)\n@@ -1,2 +1 @@\n-old content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := New()
			got := adapter.ComputeDiff(tt.baseName, tt.headName, tt.base, tt.head)

			if tt.want == "" && got != "" {
				t.Errorf("ComputeDiff() expected empty diff, got:\n%s", got)
				return
			}

			if tt.want != "" && got == "" {
				t.Errorf("ComputeDiff() expected diff, got empty")
				return
			}

			// Normalize line endings for comparison
			gotNorm := strings.ReplaceAll(got, "\r\n", "\n")
			wantNorm := strings.ReplaceAll(tt.want, "\r\n", "\n")

			if gotNorm != wantNorm {
				t.Errorf("ComputeDiff() diff mismatch:\n--- Got ---\n%s\n--- Want ---\n%s", gotNorm, wantNorm)
			}
		})
	}
}

func TestAdapter_ComputeDiff_ContextLines(t *testing.T) {
	// Test that context lines (3 before/after) are included
	adapter := New()

	base := []byte(`line1
line2
line3
line4
line5
line6
line7
line8
line9
`)
	head := []byte(`line1
line2
line3
line4
CHANGED
line6
line7
line8
line9
`)

	diff := adapter.ComputeDiff("test (main)", "test (feature)", base, head)

	// Should include 3 lines before and after the change
	if !strings.Contains(diff, "line2") { // Context before
		t.Error("Expected context line 'line2' before change")
	}
	if !strings.Contains(diff, "line8") { // Context after
		t.Error("Expected context line 'line8' after change")
	}
	if !strings.Contains(diff, "-line5") {
		t.Error("Expected removed line '-line5'")
	}
	if !strings.Contains(diff, "+CHANGED") {
		t.Error("Expected added line '+CHANGED'")
	}
}
