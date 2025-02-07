package taragen

import (
	"bytes"
	"html/template"
)

type TemplateParser struct{}

func builtinFuncs(p *Page, content []byte) template.FuncMap {
	return template.FuncMap{
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"content": func() template.HTML {
			return template.HTML(string(content))
		},
		"page": func(key string) string {
			return p.data[key].(string)
		},
		"partial": func(name string, args ...any) (template.HTML, error) {
			b, err := p.site.Partial(name, p.globals, args...)
			if err != nil {
				return "", err
			}
			return template.HTML(b), nil
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
	tmpl, err := template.New(name).Funcs(funcs).Parse(string(src))
	if err != nil {
		return nil, err
	}
	if err := tmpl.Execute(&buf, nil); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
