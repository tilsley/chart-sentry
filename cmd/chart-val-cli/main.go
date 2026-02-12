// Package main provides a CLI tool to trigger chart-val webhooks for testing.
package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"strconv"

	"github.com/google/go-github/v68/github"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := parseCliConfig()
	if err != nil {
		return err
	}

	prURL := flag.Arg(0)
	owner, repo, prNum, err := parsePRURL(prURL)
	if err != nil {
		return fmt.Errorf("parsing PR URL: %w", err)
	}

	ctx := context.Background()
	pr, err := fetchPRDetails(ctx, cfg.token, owner, repo, prNum)
	if err != nil {
		return err
	}

	payload := buildWebhookPayload(pr, owner, repo, prNum, cfg.installID)
	return sendWebhook(ctx, cfg.webhookURL, cfg.secret, payload, owner, repo, prNum, pr, prURL)
}

type cliConfig struct {
	token      string
	webhookURL string
	secret     string
	installID  int64
}

func parseCliConfig() (cliConfig, error) {
	var (
		token      = flag.String("token", "", "GitHub personal access token (or use GITHUB_TOKEN env var)")
		webhookURL = flag.String("url", "http://localhost:8080/webhook", "Webhook URL")
		secret     = flag.String(
			"secret",
			"",
			"Webhook secret for signing (read from WEBHOOK_SECRET env var if not set)",
		)
		installID = flag.Int64(
			"installation-id",
			0,
			"GitHub App installation ID (read from GITHUB_INSTALLATION_ID env var if not set)",
		)
	)
	flag.Parse()

	cfg := cliConfig{
		token:      getEnvOrFlag(*token, "GITHUB_TOKEN"),
		secret:     getEnvOrFlag(*secret, "WEBHOOK_SECRET"),
		webhookURL: *webhookURL,
	}

	if cfg.token == "" {
		return cfg, errors.New("github token required\nProvide via -token flag or GITHUB_TOKEN env var")
	}
	if cfg.secret == "" {
		return cfg, errors.New("webhook secret required\nProvide via -secret flag or WEBHOOK_SECRET env var")
	}

	cfg.installID = *installID
	if cfg.installID == 0 {
		if idStr := os.Getenv("GITHUB_INSTALLATION_ID"); idStr != "" {
			id, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil {
				return cfg, fmt.Errorf("invalid GITHUB_INSTALLATION_ID: %w", err)
			}
			cfg.installID = id
		}
	}
	if cfg.installID == 0 {
		return cfg, errors.New(
			"github App installation ID required\nProvide via -installation-id flag or GITHUB_INSTALLATION_ID env var",
		)
	}

	if flag.NArg() == 0 {
		flag.Usage()
		return cfg, errors.New("missing PR URL argument")
	}

	return cfg, nil
}

func getEnvOrFlag(flagValue, envKey string) string {
	if flagValue != "" {
		return flagValue
	}
	return os.Getenv(envKey)
}

func fetchPRDetails(ctx context.Context, token, owner, repo string, prNum int) (*github.PullRequest, error) {
	client := github.NewClient(nil).WithAuthToken(token)
	fmt.Printf("Fetching PR details from GitHub...\n")
	pr, _, err := client.PullRequests.Get(ctx, owner, repo, prNum)
	if err != nil {
		return nil, fmt.Errorf("fetching PR: %w", err)
	}
	return pr, nil
}

func buildWebhookPayload(pr *github.PullRequest, owner, repo string, prNum int, installID int64) []byte {
	payload := map[string]interface{}{
		"action": "synchronize",
		"number": prNum,
		"pull_request": map[string]interface{}{
			"number": prNum,
			"base":   map[string]interface{}{"ref": pr.GetBase().GetRef()},
			"head": map[string]interface{}{
				"ref": pr.GetHead().GetRef(),
				"sha": pr.GetHead().GetSHA(),
			},
		},
		"repository":   map[string]interface{}{"name": repo, "owner": map[string]interface{}{"login": owner}},
		"installation": map[string]interface{}{"id": installID},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		panic(fmt.Sprintf("marshaling payload: %v", err)) // Should never fail with map[string]interface{}
	}
	return payloadBytes
}

func sendWebhook(
	ctx context.Context,
	webhookURL, secret string,
	payload []byte,
	owner, repo string,
	prNum int,
	pr *github.PullRequest,
	prURL string,
) error {
	signature := signPayload(payload, secret)

	fmt.Printf("\nSending webhook to %s...\n", webhookURL)
	fmt.Printf("  Owner: %s\n", owner)
	fmt.Printf("  Repo: %s\n", repo)
	fmt.Printf("  PR: #%d\n", prNum)
	fmt.Printf("  Base: %s\n", pr.GetBase().GetRef())
	fmt.Printf("  Head: %s (%s)\n", pr.GetHead().GetRef(), pr.GetHead().GetSHA())
	fmt.Println()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Event", "pull_request")
	req.Header.Set("X-Hub-Signature-256", "sha256="+signature)
	req.Header.Set("X-GitHub-Delivery", "test-delivery-"+strconv.Itoa(prNum))

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Warn("failed to close response body", "error", err)
		}
	}()

	//nolint:errcheck // Best effort read for logging only
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusAccepted {
		fmt.Printf("✓ Webhook accepted (status %d)\n", resp.StatusCode)
		if len(body) > 0 {
			fmt.Printf("Response: %s\n", string(body))
		}
		fmt.Printf("\nCheck your GitHub PR for the results!\n")
		fmt.Printf("%s\n", prURL)
		return nil
	}

	fmt.Printf("✗ Webhook failed (status %d)\n", resp.StatusCode)
	if len(body) > 0 {
		fmt.Printf("Response: %s\n", string(body))
	}
	return fmt.Errorf("webhook returned status %d", resp.StatusCode)
}

// parsePRURL extracts owner, repo, and PR number from a GitHub PR URL
// Handles formats:
//   - https://github.com/owner/repo/pull/123
//   - https://github.com/owner/repo/pull/123/changes
//   - https://github.com/owner/repo/pull/123/files
func parsePRURL(url string) (string, string, int, error) {
	// Handle both http and https URLs, with optional trailing paths
	re := regexp.MustCompile(`github\.com/([^/]+)/([^/]+)/pull/(\d+)(?:/.*)?$`)
	matches := re.FindStringSubmatch(url)

	if len(matches) != 4 {
		return "", "", 0, fmt.Errorf(
			"invalid PR URL format, expected: https://github.com/owner/repo/pull/123, got: %s",
			url,
		)
	}

	owner := matches[1]
	repo := matches[2]
	prNum, err := strconv.Atoi(matches[3])
	if err != nil {
		return "", "", 0, fmt.Errorf("invalid PR number: %w", err)
	}

	return owner, repo, prNum, nil
}

// signPayload creates HMAC SHA256 signature for the payload
func signPayload(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}
