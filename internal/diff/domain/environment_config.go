package domain

// EnvironmentConfig holds the specific environment context and
// the ordered list of values file paths (Helm applies left-to-right).
type EnvironmentConfig struct {
	Name       string
	ValueFiles []string
}
