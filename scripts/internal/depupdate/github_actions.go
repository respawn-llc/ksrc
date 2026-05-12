package depupdate

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

type GitHubActionsOptions struct {
	Client     *http.Client
	APIBaseURL string
	Token      string
}

type semver struct {
	major int
	minor int
	patch int
}

type tag struct {
	Name string `json:"name"`
}

var (
	usesPattern = regexp.MustCompile(`(?m)^(\s*(?:-\s+)?uses:\s+)([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+(?:/[A-Za-z0-9_.-]+)*)@([^[:space:]#]+)(.*)$`)
	tagPattern  = regexp.MustCompile(`^v?([0-9]+)(?:\.([0-9]+))?(?:\.([0-9]+))?$`)
)

func UpdateGitHubActions(workflowDir string, options GitHubActionsOptions) ([]string, error) {
	files, err := workflowFiles(workflowDir)
	if err != nil {
		return nil, err
	}
	contents := make(map[string]string, len(files))
	repos := map[string]struct{}{}
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		text := string(content)
		contents[file] = text
		for _, matches := range usesPattern.FindAllStringSubmatch(text, -1) {
			if _, ok := parseSemver(matches[3]); ok {
				repos[repoFromAction(matches[2])] = struct{}{}
			}
		}
	}

	repoNames := make([]string, 0, len(repos))
	for repo := range repos {
		repoNames = append(repoNames, repo)
	}
	slices.Sort(repoNames)

	client := options.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	baseURL := options.APIBaseURL
	if baseURL == "" {
		baseURL = "https://api.github.com"
	}

	latestByRepo := make(map[string]string, len(repoNames))
	var failures []string
	for _, repo := range repoNames {
		latest, err := latestStableTag(client, strings.TrimRight(baseURL, "/"), repo, options.Token)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", repo, err))
			continue
		}
		latestByRepo[repo] = latest
	}
	if len(failures) > 0 {
		return nil, fmt.Errorf("failed to resolve latest GitHub Action tags (set GITHUB_TOKEN if rate limited): %s", strings.Join(failures, "; "))
	}

	updatedByFile := make(map[string]string, len(files))
	var updates []string
	for _, file := range files {
		updated := usesPattern.ReplaceAllStringFunc(contents[file], func(line string) string {
			matches := usesPattern.FindStringSubmatch(line)
			if len(matches) == 0 {
				return line
			}
			current := matches[3]
			if _, ok := parseSemver(current); !ok {
				return line
			}
			latest := latestByRepo[repoFromAction(matches[2])]
			if latest == "" || latest == current {
				return line
			}
			updates = append(updates, fmt.Sprintf("%s: %s %s -> %s", file, matches[2], current, latest))
			return matches[1] + matches[2] + "@" + latest + matches[4]
		})
		updatedByFile[file] = updated
	}
	for _, file := range files {
		if updatedByFile[file] == contents[file] {
			continue
		}
		if err := os.WriteFile(file, []byte(updatedByFile[file]), 0o644); err != nil {
			return nil, err
		}
	}
	return updates, nil
}

func workflowFiles(dir string) ([]string, error) {
	var files []string
	if err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		switch filepath.Ext(path) {
		case ".yml", ".yaml":
			files = append(files, path)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	slices.Sort(files)
	return files, nil
}

func repoFromAction(action string) string {
	parts := strings.Split(action, "/")
	if len(parts) < 2 {
		return action
	}
	return parts[0] + "/" + parts[1]
}

func latestStableTag(client *http.Client, baseURL string, repo string, token string) (string, error) {
	var bestName string
	var best semver
	for page := 1; page <= 10; page++ {
		url := fmt.Sprintf("%s/repos/%s/tags?per_page=100&page=%d", baseURL, repo, page)
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return "", err
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("User-Agent", "ksrc-dependency-updater")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		body, readErr := io.ReadAll(resp.Body)
		closeErr := resp.Body.Close()
		if readErr != nil {
			return "", readErr
		}
		if closeErr != nil {
			return "", closeErr
		}
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(body)))
		}
		var tags []tag
		if err := json.Unmarshal(body, &tags); err != nil {
			return "", err
		}
		if len(tags) == 0 {
			break
		}
		for _, tag := range tags {
			version, ok := parseSemver(tag.Name)
			if !ok {
				continue
			}
			if bestName == "" || compareSemver(version, best) > 0 {
				bestName = tag.Name
				best = version
			}
		}
	}
	if bestName == "" {
		return "", errors.New("no stable semver tags found")
	}
	return bestName, nil
}

func parseSemver(raw string) (semver, bool) {
	matches := tagPattern.FindStringSubmatch(raw)
	if len(matches) == 0 {
		return semver{}, false
	}
	major, err := strconv.Atoi(matches[1])
	if err != nil {
		return semver{}, false
	}
	minor := 0
	if matches[2] != "" {
		var err error
		minor, err = strconv.Atoi(matches[2])
		if err != nil {
			return semver{}, false
		}
	}
	patch := 0
	if matches[3] != "" {
		var err error
		patch, err = strconv.Atoi(matches[3])
		if err != nil {
			return semver{}, false
		}
	}
	return semver{major: major, minor: minor, patch: patch}, true
}

func compareSemver(left semver, right semver) int {
	switch {
	case left.major != right.major:
		return left.major - right.major
	case left.minor != right.minor:
		return left.minor - right.minor
	default:
		return left.patch - right.patch
	}
}
