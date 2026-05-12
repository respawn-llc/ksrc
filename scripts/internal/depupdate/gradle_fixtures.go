package depupdate

import (
	"fmt"
	"os"
	"regexp"
)

var (
	catalogDatetimeVersionPattern = regexp.MustCompile(`(?m)^kotlinx-datetime\s*=\s*"([^"]+)"\s*$`)
	fixtureDatetimeDepPattern     = regexp.MustCompile(`org\.jetbrains\.kotlinx:kotlinx-datetime:[^'"]+`)
)

func SyncGradleFixtureDeps(catalogPath string, fixturePath string) error {
	catalog, err := os.ReadFile(catalogPath)
	if err != nil {
		return err
	}
	version, err := KotlinxDatetimeVersion(catalogPath, string(catalog))
	if err != nil {
		return err
	}
	fixture, err := os.ReadFile(fixturePath)
	if err != nil {
		return err
	}
	updated := fixtureDatetimeDepPattern.ReplaceAll(fixture, []byte("org.jetbrains.kotlinx:kotlinx-datetime:"+version))
	if string(updated) == string(fixture) {
		return nil
	}
	return os.WriteFile(fixturePath, updated, 0o644)
}

func KotlinxDatetimeVersion(sourceName string, catalog string) (string, error) {
	versionMatch := catalogDatetimeVersionPattern.FindStringSubmatch(catalog)
	if len(versionMatch) == 0 {
		return "", fmt.Errorf("%s: missing kotlinx-datetime version", sourceName)
	}
	return versionMatch[1], nil
}
