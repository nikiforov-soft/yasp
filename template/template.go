package template

import (
	"bytes"
	"fmt"
	"text/template"
)

func Execute(templateKey, templateValue string, data any) ([]byte, error) {
	tmpl, err := template.New(templateKey).Funcs(funcsMap).Parse(templateValue)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute %s template: - %w", templateKey, err)
	}
	return buf.Bytes(), nil
}
