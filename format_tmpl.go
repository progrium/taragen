package taragen

import (
	"bytes"
	"text/template"
)

type TemplateParser struct{}

func builtinFuncs(p *Page, content []byte) template.FuncMap {
	return template.FuncMap{
		"safeHTML": func(s string) string {
			return s
		},
		"content": func() string {
			return string(content)
		},
		"page": func(key string) string {
			return p.data[key].(string)
		},
		"partial": func(name string, args ...any) (string, error) {
			b, err := p.site.Partial(name, p.globals, args...)
			if err != nil {
				return "", err
			}
			return string(b), nil
		},
	}
}

func (f *TemplateParser) Parse(p *Page) ([]byte, Data, error) {
	data, rest, err := SplitFrontmatter(p.Source())
	if err != nil {
		return nil, nil, err
	}

	b, err := RenderTemplate(p.path, rest, builtinFuncs(p, p.Body()))
	if err != nil {
		return nil, nil, err
	}
	return b, data, nil
}

func RenderTemplate(name string, src []byte, funcs template.FuncMap) ([]byte, error) {
	var buf bytes.Buffer
	tmpl, err := template.New(name).Funcs(funcs).Delims("[[", "]]").Parse(string(src))
	if err != nil {
		return nil, err
	}
	if err := tmpl.Execute(&buf, nil); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
