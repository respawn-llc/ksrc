package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestSkillMentionsEveryCommand(t *testing.T) {
	link, err := os.Readlink(filepath.Join("..", "..", "skills", "ksrc"))
	if err != nil {
		t.Fatalf("skills/ksrc symlink: %v", err)
	}
	if link != "../plugins/ksrc/skills/ksrc" {
		t.Fatalf("skills/ksrc -> %q, want ../plugins/ksrc/skills/ksrc", link)
	}
	body, err := os.ReadFile(filepath.Join("..", "..", "skills", "ksrc", "SKILL.md"))
	if err != nil {
		t.Fatalf("read skill: %v", err)
	}
	text := string(body)
	description := skillDescription(t, text)
	if !strings.Contains(description, "dependency sources") {
		t.Fatal("skill description missing dependency-source trigger")
	}
	if !strings.Contains(description, "this repository's own source") {
		t.Fatal("skill description missing own-source trigger")
	}
	if len(description) > 1024 {
		t.Fatalf("skill description length %d exceeds 1024", len(description))
	}

	skip := map[string]struct{}{"help": {}, "completion": {}}
	var missing []string
	for _, cmd := range NewRootCommand(NewApp()).Commands() {
		for _, name := range append([]string{cmd.Name()}, cmd.Aliases...) {
			if _, ok := skip[name]; ok {
				continue
			}
			if !mentionsIdent(text, name) {
				missing = append(missing, name)
			}
		}
	}
	if len(missing) > 0 {
		t.Fatalf("skills/ksrc/SKILL.md missing commands: %s", strings.Join(missing, ", "))
	}
}

func skillDescription(t *testing.T, text string) string {
	t.Helper()
	const fence = "---"
	start := strings.Index(text, fence)
	if start != 0 {
		t.Fatal("skill missing opening frontmatter")
	}
	rest := text[len(fence):]
	end := strings.Index(rest, fence)
	if end < 0 {
		t.Fatal("skill missing closing frontmatter")
	}
	front := rest[:end]
	const key = "description:"
	i := strings.Index(front, key)
	if i < 0 {
		t.Fatal("skill missing description")
	}
	value := strings.TrimSpace(front[i+len(key):])
	value = strings.Trim(value, `"`)
	if value == "" {
		t.Fatal("skill description is empty")
	}
	return value
}

func mentionsIdent(text, name string) bool {
	re := regexp.MustCompile(`(?:^|[^A-Za-z0-9_])` + regexp.QuoteMeta(name) + `(?:[^A-Za-z0-9_]|$)`)
	return re.MatchString(text)
}
