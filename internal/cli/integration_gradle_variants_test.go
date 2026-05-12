package cli

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIntegrationTargetsNarrowExternalVariantSources(t *testing.T) {
	if os.Getenv("KSRC_INTEGRATION") != "1" {
		t.Skip("set KSRC_INTEGRATION=1 to run")
	}

	projectDir := prepareVariantTargetProject(t)

	all := runVariantTargetResolve(t, projectDir)
	assertResolveContains(t, all, "com.example:demo:1.0")
	assertResolveContains(t, all, "com.example:demo-jvm:1.0")
	assertResolveContains(t, all, "com.example:demo-js:1.0")
	assertNoDuplicateResolvePaths(t, all)

	jvm := runVariantTargetResolve(t, projectDir, "--targets", "jvm")
	assertResolveContains(t, jvm, "com.example:demo:1.0")
	assertResolveContains(t, jvm, "com.example:demo-jvm:1.0")
	assertResolveExcludes(t, jvm, "com.example:demo-js:1.0")

	js := runVariantTargetResolve(t, projectDir, "--targets", "js")
	assertResolveContains(t, js, "com.example:demo:1.0")
	assertResolveContains(t, js, "com.example:demo-js:1.0")
	assertResolveExcludes(t, js, "com.example:demo-jvm:1.0")
}

func runVariantTargetResolve(t *testing.T, projectDir string, extraArgs ...string) string {
	t.Helper()

	args := []string{"resolve", "--module", "com.example:demo", "--project", projectDir}
	args = append(args, extraArgs...)
	out, err := runCommand(NewApp(), args)
	if err != nil {
		t.Fatalf("resolve error: %v\n%s", err, out)
	}
	return out
}

func assertResolveContains(t *testing.T, out string, coord string) {
	t.Helper()

	if !strings.Contains(out, coord+"|") {
		t.Fatalf("expected %s in resolve output:\n%s", coord, out)
	}
}

func assertResolveExcludes(t *testing.T, out string, coord string) {
	t.Helper()

	if strings.Contains(out, coord+"|") {
		t.Fatalf("did not expect %s in resolve output:\n%s", coord, out)
	}
}

func prepareVariantTargetProject(t *testing.T) string {
	t.Helper()

	dst := t.TempDir()
	copyGradleWrapper(t, dst)
	repo := filepath.Join(dst, "repo")
	writeVariantTargetRepo(t, repo)
	writeTextFile(t, filepath.Join(dst, "gradle.properties"), strings.Join([]string{
		"org.gradle.configuration-cache=true",
		"org.gradle.unsafe.isolated-projects=true",
		"",
	}, "\n"))
	writeTextFile(t, filepath.Join(dst, "settings.gradle"), "rootProject.name = 'ksrc-variant-targets'\n")
	writeTextFile(t, filepath.Join(dst, "build.gradle"), fmt.Sprintf(`
import org.gradle.api.attributes.Attribute
import org.gradle.api.attributes.Category
import org.gradle.api.attributes.Usage
import org.gradle.api.attributes.LibraryElements
import org.gradle.api.attributes.java.TargetJvmEnvironment
import org.gradle.api.attributes.java.TargetJvmVersion

def kotlinPlatformType = Attribute.of('org.jetbrains.kotlin.platform.type', String)
def kotlinJsCompiler = Attribute.of('org.jetbrains.kotlin.js.compiler', String)

repositories {
    maven {
        url = uri('%s')
        metadataSources {
            gradleMetadata()
            mavenPom()
        }
    }
}

configurations {
    jvmMainCompileClasspath {
        canBeResolved = true
        canBeConsumed = false
        attributes {
            attribute(Category.CATEGORY_ATTRIBUTE, objects.named(Category, Category.LIBRARY))
            attribute(Usage.USAGE_ATTRIBUTE, objects.named(Usage, Usage.JAVA_API))
            attribute(LibraryElements.LIBRARY_ELEMENTS_ATTRIBUTE, objects.named(LibraryElements, LibraryElements.JAR))
            attribute(TargetJvmEnvironment.TARGET_JVM_ENVIRONMENT_ATTRIBUTE, objects.named(TargetJvmEnvironment, TargetJvmEnvironment.STANDARD_JVM))
            attribute(TargetJvmVersion.TARGET_JVM_VERSION_ATTRIBUTE, 8)
            attribute(kotlinPlatformType, 'jvm')
        }
    }
    jsMainCompileClasspath {
        canBeResolved = true
        canBeConsumed = false
        attributes {
            attribute(Category.CATEGORY_ATTRIBUTE, objects.named(Category, Category.LIBRARY))
            attribute(Usage.USAGE_ATTRIBUTE, objects.named(Usage, 'kotlin-api'))
            attribute(TargetJvmEnvironment.TARGET_JVM_ENVIRONMENT_ATTRIBUTE, objects.named(TargetJvmEnvironment, 'non-jvm'))
            attribute(kotlinPlatformType, 'js')
            attribute(kotlinJsCompiler, 'ir')
        }
    }
}

dependencies {
    jvmMainCompileClasspath 'com.example:demo:1.0'
    jsMainCompileClasspath 'com.example:demo:1.0'
}
`, filepath.ToSlash(repo)))
	return dst
}

func writeVariantTargetRepo(t *testing.T, repo string) {
	t.Helper()

	writeVariantModule(t, repo, "demo", map[string]string{"commonMain/Demo.kt": "common source\n"}, baseVariantMetadata())
	writeVariantModule(t, repo, "demo-jvm", map[string]string{"jvmMain/DemoJvm.kt": "jvm source\n"}, externalVariantMetadata("demo-jvm", jvmVariantAttributes()))
	writeVariantModule(t, repo, "demo-js", map[string]string{"jsMain/DemoJs.kt": "js source\n"}, externalVariantMetadata("demo-js", jsVariantAttributes()))
}

func writeVariantModule(t *testing.T, repo string, artifact string, sourceEntries map[string]string, metadata gradleModuleMetadata) {
	t.Helper()

	dir := filepath.Join(repo, "com", "example", artifact, "1.0")
	writeZipFile(t, filepath.Join(dir, artifact+"-1.0.jar"), map[string]string{"artifact.txt": artifact + "\n"})
	writeZipFile(t, filepath.Join(dir, artifact+"-1.0-sources.jar"), sourceEntries)
	writeTextFile(t, filepath.Join(dir, artifact+"-1.0.pom"), fmt.Sprintf(
		"<project><modelVersion>4.0.0</modelVersion><groupId>com.example</groupId><artifactId>%s</artifactId><version>1.0</version></project>\n",
		artifact,
	))
	encoded, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		t.Fatalf("marshal module metadata for %s: %v", artifact, err)
	}
	writeTextFile(t, filepath.Join(dir, artifact+"-1.0.module"), string(encoded)+"\n")
}

func baseVariantMetadata() gradleModuleMetadata {
	return gradleModuleMetadata{
		FormatVersion: "1.1",
		Component:     componentMetadata("demo"),
		Variants: []gradleModuleVariant{
			{
				Name:       "metadataApiElements",
				Attributes: map[string]any{"org.gradle.category": "library", "org.gradle.usage": "kotlin-metadata", "org.jetbrains.kotlin.platform.type": "common"},
				Files:      []gradleModuleFile{{Name: "demo-1.0.jar", URL: "demo-1.0.jar"}},
			},
			{
				Name:        "jvmApiElements-published",
				Attributes:  jvmVariantAttributes(),
				AvailableAt: availableAtMetadata("demo-jvm"),
			},
			{
				Name:        "jsApiElements-published",
				Attributes:  jsVariantAttributes(),
				AvailableAt: availableAtMetadata("demo-js"),
			},
		},
	}
}

func externalVariantMetadata(artifact string, attributes map[string]any) gradleModuleMetadata {
	return gradleModuleMetadata{
		FormatVersion: "1.1",
		Component:     componentMetadata(artifact),
		Variants: []gradleModuleVariant{
			{
				Name:       "apiElements",
				Attributes: attributes,
				Files:      []gradleModuleFile{{Name: artifact + "-1.0.jar", URL: artifact + "-1.0.jar"}},
			},
		},
	}
}

type gradleModuleMetadata struct {
	FormatVersion string                `json:"formatVersion"`
	Component     gradleModuleComponent `json:"component"`
	Variants      []gradleModuleVariant `json:"variants"`
}

type gradleModuleComponent struct {
	Group   string `json:"group"`
	Module  string `json:"module"`
	Version string `json:"version"`
}

type gradleModuleVariant struct {
	Name        string                   `json:"name"`
	Attributes  map[string]any           `json:"attributes,omitempty"`
	Files       []gradleModuleFile       `json:"files,omitempty"`
	AvailableAt *gradleModuleAvailableAt `json:"available-at,omitempty"`
}

type gradleModuleFile struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type gradleModuleAvailableAt struct {
	URL     string `json:"url"`
	Group   string `json:"group"`
	Module  string `json:"module"`
	Version string `json:"version"`
}

func componentMetadata(artifact string) gradleModuleComponent {
	return gradleModuleComponent{Group: "com.example", Module: artifact, Version: "1.0"}
}

func availableAtMetadata(artifact string) *gradleModuleAvailableAt {
	return &gradleModuleAvailableAt{URL: "../../" + artifact + "/1.0/" + artifact + "-1.0.module", Group: "com.example", Module: artifact, Version: "1.0"}
}

func jvmVariantAttributes() map[string]any {
	return map[string]any{
		"org.gradle.category":                "library",
		"org.gradle.jvm.environment":         "standard-jvm",
		"org.gradle.jvm.version":             8,
		"org.gradle.libraryelements":         "jar",
		"org.gradle.usage":                   "java-api",
		"org.jetbrains.kotlin.platform.type": "jvm",
	}
}

func jsVariantAttributes() map[string]any {
	return map[string]any{
		"org.gradle.category":                "library",
		"org.gradle.jvm.environment":         "non-jvm",
		"org.gradle.usage":                   "kotlin-api",
		"org.jetbrains.kotlin.js.compiler":   "ir",
		"org.jetbrains.kotlin.platform.type": "js",
	}
}

func writeZipFile(t *testing.T, path string, entries map[string]string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create zip %s: %v", path, err)
	}
	zipWriter := zip.NewWriter(file)
	for name, contents := range entries {
		writer, err := zipWriter.Create(name)
		if err != nil {
			t.Fatalf("create zip entry %s: %v", name, err)
		}
		if _, err := writer.Write([]byte(contents)); err != nil {
			t.Fatalf("write zip entry %s: %v", name, err)
		}
	}
	if err := zipWriter.Close(); err != nil {
		t.Fatalf("close zip writer %s: %v", path, err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close zip %s: %v", path, err)
	}
}
