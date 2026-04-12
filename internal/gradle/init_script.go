package gradle

import (
	"bytes"
	_ "embed"
	"fmt"
	"text/template"

	"github.com/respawn-app/ksrc/internal/resolve"
)

const initScriptTemplateVersion = "v1"

//go:embed templates/init_script.v1.gradle.tmpl
var initScriptTemplateSource string

var initScriptTemplate = template.Must(template.New("init_script." + initScriptTemplateVersion + ".gradle.tmpl").Option("missingkey=error").Parse(initScriptTemplateSource))

type initScriptTemplateData struct {
	SelectorHelpers string
}

func InitScript() string {
	script, err := renderInitScript(initScriptTemplateData{SelectorHelpers: resolve.GradleSelectorHelpers()})
	if err != nil {
		panic(fmt.Sprintf("render init script template %s: %v", initScriptTemplateVersion, err))
	}
	return script
}

func renderInitScript(data initScriptTemplateData) (string, error) {
	var script bytes.Buffer
	if err := initScriptTemplate.Execute(&script, data); err != nil {
		return "", err
	}
	return script.String(), nil
}
