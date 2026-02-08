package domain

import "strings"

// ExtractChartNames parses file paths and returns unique chart names
// from paths matching charts/{name}/...
// This encapsulates the repository structure convention.
func ExtractChartNames(files []string) []string {
	seen := make(map[string]struct{})
	var names []string
	for _, f := range files {
		if !strings.HasPrefix(f, "charts/") {
			continue
		}
		// charts/{name}/... → extract {name}
		parts := strings.SplitN(f, "/", 3)
		if len(parts) < 2 || parts[1] == "" {
			continue
		}
		name := parts[1]
		if _, ok := seen[name]; !ok {
			seen[name] = struct{}{}
			names = append(names, name)
		}
	}
	return names
}
