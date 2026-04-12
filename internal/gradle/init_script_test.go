package gradle

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/respawn-app/ksrc/internal/resolve"
)

type selectorProbeCoord struct {
	Group    string `json:"group"`
	Artifact string `json:"artifact"`
	Version  string `json:"version"`
}

type selectorProbeCase struct {
	Name     string             `json:"name"`
	Module   string             `json:"module"`
	Group    string             `json:"group"`
	Artifact string             `json:"artifact"`
	Version  string             `json:"version"`
	Coord    selectorProbeCoord `json:"coord"`
}

type globProbeCase struct {
	Name     string `json:"name"`
	Patterns string `json:"patterns"`
	Value    string `json:"value"`
}

type selectorProbeInput struct {
	SelectorCases []selectorProbeCase `json:"selectorCases"`
	GlobCases     []globProbeCase     `json:"globCases"`
}

type selectorProbeOutput struct {
	Selector []bool `json:"selector"`
	Glob     []bool `json:"glob"`
}

func TestInitScriptUsesSharedSelectorHelpers(t *testing.T) {
	t.Parallel()

	script := InitScript()
	checks := []string{
		"def matchesCoordinateSelectors =",
		"def moduleSelectorCandidates =",
		"matchesCoordinateSelectors(moduleProp as String, groupProp as String, artifactProp as String, versionProp as String, id.group, id.module, id.version)",
		"selector.split(':', -1)",
	}
	for _, check := range checks {
		if !strings.Contains(script, check) {
			t.Fatalf("expected init script to contain %q", check)
		}
	}
}

func TestGradleSelectorHelpersMatchGoSemantics(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("gradle"); err != nil {
		t.Skip("gradle not available")
	}

	input := selectorProbeInput{
		SelectorCases: []selectorProbeCase{
			{
				Name:   "qualified trailing colon",
				Module: "org.jetbrains.kotlinx:kotlinx-datetime:",
				Coord:  selectorProbeCoord{Group: "org.jetbrains.kotlinx", Artifact: "kotlinx-datetime", Version: "0.7.1"},
			},
			{
				Name:   "loose normalized artifact",
				Module: "kotlinx.datetime",
				Coord:  selectorProbeCoord{Group: "org.jetbrains.kotlinx", Artifact: "kotlinx-datetime", Version: "0.7.1"},
			},
			{
				Name:   "loose normalized jvm variant",
				Module: "kotlinx.datetime",
				Coord:  selectorProbeCoord{Group: "org.jetbrains.kotlinx", Artifact: "kotlinx-datetime-jvm", Version: "0.7.1"},
			},
			{
				Name:     "combined selectors narrow match",
				Module:   "org.jetbrains.kotlinx:*datetime*:",
				Group:    "org.jetbrains.*",
				Artifact: "*datetime",
				Version:  "0.*",
				Coord:    selectorProbeCoord{Group: "org.jetbrains.kotlinx", Artifact: "kotlinx-datetime", Version: "0.7.1"},
			},
			{
				Name:     "combined selectors reject different version",
				Module:   "org.jetbrains.kotlinx:*datetime*:",
				Group:    "org.jetbrains.*",
				Artifact: "*datetime",
				Version:  "0.*",
				Coord:    selectorProbeCoord{Group: "org.jetbrains.kotlinx", Artifact: "kotlinx-datetime", Version: "1.0.0"},
			},
		},
		GlobCases: []globProbeCase{
			{Name: "slash segment match", Patterns: "commonMain/*", Value: "commonMain/LocalDate.kt"},
			{Name: "slash segment reject nested", Patterns: "commonMain/*", Value: "commonMain/internal/Platform.kt"},
			{Name: "escaped literal star", Patterns: "literal\\*value", Value: "literal*value"},
			{Name: "question mark", Patterns: "module-?.jar", Value: "module-a.jar"},
			{Name: "char class", Patterns: "file[0-9].kt", Value: "file7.kt"},
			{Name: "slash and backslash differ", Patterns: "commonMain/LocalDate.kt", Value: "commonMain\\LocalDate.kt"},
			{Name: "invalid glob exact fallback", Patterns: "file[.kt", Value: "file[.kt"},
		},
	}

	got := runGradleSelectorProbe(t, input)

	if len(got.Selector) != len(input.SelectorCases) {
		t.Fatalf("selector result count = %d, want %d", len(got.Selector), len(input.SelectorCases))
	}
	for i, testCase := range input.SelectorCases {
		want := resolve.NewSelector(testCase.Module, testCase.Group, testCase.Artifact, testCase.Version).MatchCoord(resolve.Coord{
			Group:    testCase.Coord.Group,
			Artifact: testCase.Coord.Artifact,
			Version:  testCase.Coord.Version,
		})
		if got.Selector[i] != want {
			t.Fatalf("selector case %q mismatch: got %v want %v", testCase.Name, got.Selector[i], want)
		}
	}

	if len(got.Glob) != len(input.GlobCases) {
		t.Fatalf("glob result count = %d, want %d", len(got.Glob), len(input.GlobCases))
	}
	for i, testCase := range input.GlobCases {
		want := resolve.MatchAny(testCase.Patterns, testCase.Value)
		if got.Glob[i] != want {
			t.Fatalf("glob case %q mismatch: got %v want %v", testCase.Name, got.Glob[i], want)
		}
	}
}

func runGradleSelectorProbe(t *testing.T, input selectorProbeInput) selectorProbeOutput {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "settings.gradle"), []byte("rootProject.name = 'selector-probe'\n"), 0o644); err != nil {
		t.Fatalf("write settings.gradle: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "build.gradle"), []byte("\n"), 0o644); err != nil {
		t.Fatalf("write build.gradle: %v", err)
	}

	probePath := filepath.Join(dir, "probe.json")
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal probe input: %v", err)
	}
	if err := os.WriteFile(probePath, data, 0o644); err != nil {
		t.Fatalf("write probe.json: %v", err)
	}

	scriptPath := filepath.Join(dir, "selector-probe.init.gradle")
	script := fmt.Sprintf(`
import groovy.json.JsonOutput
import groovy.json.JsonSlurper
%s

def probe = new JsonSlurper().parse(new File(%q))

gradle.rootProject { root ->
    root.tasks.register('selectorProbe') {
        doLast {
            def result = [
                selector: probe.selectorCases.collect { testCase ->
                    matchesCoordinateSelectors(
                        testCase.module as String,
                        testCase.group as String,
                        testCase.artifact as String,
                        testCase.version as String,
                        testCase.coord.group as String,
                        testCase.coord.artifact as String,
                        testCase.coord.version as String
                    )
                },
                glob: probe.globCases.collect { testCase ->
                    matchesGlobPattern(testCase.patterns as String, testCase.value as String)
                }
            ]
            println(JsonOutput.toJson(result))
        }
    }
}
`, resolve.GradleSelectorHelpers(), probePath)
	if err := os.WriteFile(scriptPath, []byte(script), 0o644); err != nil {
		t.Fatalf("write init script: %v", err)
	}

	cmd := exec.Command("gradle", "-q", "--no-daemon", "--no-configuration-cache", "-I", scriptPath, "selectorProbe")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run gradle probe: %v\n%s", err, string(out))
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var result selectorProbeOutput
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &result); err != nil {
		t.Fatalf("parse gradle probe output: %v\n%s", err, string(out))
	}
	return result
}
