package depupdate

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateGitHubActionsUpdatesStableTags(t *testing.T) {
	t.Parallel()

	server := newTagServer(t, map[string]string{
		"actions/checkout": `[{"name":"v2.0.0-beta"},{"name":"v2.1.0"},{"name":"v1.9.0"}]`,
		"owner/tool":       `[{"name":"v3.0.1"},{"name":"v2.9.0"}]`,
	})
	workflowDir := t.TempDir()
	workflow := filepath.Join(workflowDir, "ci.yml")
	writeFile(t, workflow, ""+
		"steps:\n"+
		"  - uses: actions/checkout@v1.0.0\n"+
		"  - name: tool\n"+
		"    uses: owner/tool/subpath@v2.0.0\n"+
		"  - uses: owner/not-semver@main\n")

	updates, err := UpdateGitHubActions(workflowDir, GitHubActionsOptions{
		Client:     server.Client(),
		APIBaseURL: server.URL,
	})
	if err != nil {
		t.Fatalf("UpdateGitHubActions error: %v", err)
	}
	got := readFile(t, workflow)
	if !strings.Contains(got, "actions/checkout@v2.1.0") {
		t.Fatalf("checkout was not updated:\n%s", got)
	}
	if !strings.Contains(got, "owner/tool/subpath@v3.0.1") {
		t.Fatalf("subpath action was not updated:\n%s", got)
	}
	if !strings.Contains(got, "owner/not-semver@main") {
		t.Fatalf("non-semver action was changed:\n%s", got)
	}
	if len(updates) != 2 {
		t.Fatalf("expected 2 updates, got %d: %v", len(updates), updates)
	}
}

func TestUpdateGitHubActionsDoesNotWriteWhenResolutionFails(t *testing.T) {
	t.Parallel()

	server := newTagServer(t, map[string]string{
		"actions/checkout": `[{"name":"v2.0.0"}]`,
	})
	workflowDir := t.TempDir()
	workflow := filepath.Join(workflowDir, "ci.yml")
	original := "" +
		"steps:\n" +
		"  - uses: actions/checkout@v1.0.0\n" +
		"  - uses: missing/action@v1.0.0\n"
	writeFile(t, workflow, original)

	_, err := UpdateGitHubActions(workflowDir, GitHubActionsOptions{
		Client:     server.Client(),
		APIBaseURL: server.URL,
	})
	if err == nil {
		t.Fatal("expected resolution error")
	}
	if got := readFile(t, workflow); got != original {
		t.Fatalf("workflow changed despite failed resolution:\n%s", got)
	}
}

func newTagServer(t *testing.T, tagsByRepo map[string]string) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		repo := strings.TrimPrefix(r.URL.Path, "/repos/")
		repo = strings.TrimSuffix(repo, "/tags")
		tags, ok := tagsByRepo[repo]
		if !ok {
			http.Error(w, fmt.Sprintf("missing repo %s", repo), http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(tags))
	}))
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(content)
}
