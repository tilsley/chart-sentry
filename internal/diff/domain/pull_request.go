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
