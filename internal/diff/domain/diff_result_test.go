package domain

import "testing"

func TestDiffResult_PreferredDiff(t *testing.T) {
	tests := []struct {
		name         string
		result       DiffResult
		wantContains string
	}{
		{
			name: "prefers semantic diff when available",
			result: DiffResult{
				SemanticDiff: "semantic diff content",
				UnifiedDiff:  "unified diff content",
			},
			wantContains: "semantic",
		},
		{
			name: "falls back to unified diff when semantic is empty",
			result: DiffResult{
				SemanticDiff: "",
				UnifiedDiff:  "unified diff content",
			},
			wantContains: "unified",
		},
		{
			name: "returns empty when both are empty",
			result: DiffResult{
				SemanticDiff: "",
				UnifiedDiff:  "",
			},
			wantContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.result.PreferredDiff()
			if tt.wantContains != "" && got != tt.result.SemanticDiff && got != tt.result.UnifiedDiff {
				t.Errorf("PreferredDiff() = %v, want one of semantic or unified", got)
			}
			if tt.wantContains == "" && got != "" {
				t.Errorf("PreferredDiff() = %v, want empty", got)
			}
		})
	}
}

func TestCountByStatus(t *testing.T) {
	tests := []struct {
		name         string
		results      []DiffResult
		wantSuccess  int
		wantChanges  int
		wantErrors   int
	}{
		{
			name:        "empty results",
			results:     []DiffResult{},
			wantSuccess: 0,
			wantChanges: 0,
			wantErrors:  0,
		},
		{
			name: "all success",
			results: []DiffResult{
				{Status: StatusSuccess},
				{Status: StatusSuccess},
			},
			wantSuccess: 2,
			wantChanges: 0,
			wantErrors:  0,
		},
		{
			name: "all changes",
			results: []DiffResult{
				{Status: StatusChanges},
				{Status: StatusChanges},
			},
			wantSuccess: 0,
			wantChanges: 2,
			wantErrors:  0,
		},
		{
			name: "all errors",
			results: []DiffResult{
				{Status: StatusError},
				{Status: StatusError},
			},
			wantSuccess: 0,
			wantChanges: 0,
			wantErrors:  2,
		},
		{
			name: "mixed statuses",
			results: []DiffResult{
				{Status: StatusSuccess},
				{Status: StatusChanges},
				{Status: StatusError},
				{Status: StatusSuccess},
				{Status: StatusChanges},
			},
			wantSuccess: 2,
			wantChanges: 2,
			wantErrors:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSuccess, gotChanges, gotErrors := CountByStatus(tt.results)
			if gotSuccess != tt.wantSuccess {
				t.Errorf("CountByStatus() success = %v, want %v", gotSuccess, tt.wantSuccess)
			}
			if gotChanges != tt.wantChanges {
				t.Errorf("CountByStatus() changes = %v, want %v", gotChanges, tt.wantChanges)
			}
			if gotErrors != tt.wantErrors {
				t.Errorf("CountByStatus() errors = %v, want %v", gotErrors, tt.wantErrors)
			}
		})
	}
}
