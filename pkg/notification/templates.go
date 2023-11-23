package notification

import (
	"bytes"
	"html/template"
	"os"
)

func FileTemplate(path string) *template.Template {
	raw, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}

	var tmpl *template.Template
	if tmpl, err = template.New(path).Parse(string(raw)); err != nil {
		panic(err)
	}

	return tmpl
}

func ExecuteTemplate(t *template.Template, data interface{}) *bytes.Buffer {
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		panic(err)
	}

	return &buf
}
