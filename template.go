package taragen

import (
	"bytes"
	"html/template"
)

type TemplateFormat struct{}

func (f *TemplateFormat) Parse(p *Page) ([]byte, Data, error) {
	data, rest, err := SplitFrontmatter(p.Source())
	if err != nil {
		return nil, nil, err
	}

	b, err := RenderTemplate(p.path, rest, p.templateFuncs())
	if err != nil {
		return nil, nil, err
	}
	return b, data, nil
}

func RenderTemplate(name string, src []byte, funcs template.FuncMap) ([]byte, error) {
	var buf bytes.Buffer
	tmpl, err := template.New(name).Funcs(funcs).Parse(string(src))
	if err != nil {
		return nil, err
	}
	if err := tmpl.Execute(&buf, nil); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
