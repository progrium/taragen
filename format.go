package taragen

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v2"
)

const (
	ExtMarkdown = ".md"
	ExtTemplate = ".tmpl"
	ExtJSX      = ".jsx"
)

var Formats = map[string]PageParser{
	ExtMarkdown: &MarkdownParser{},
	ExtTemplate: &TemplateParser{},
	ExtJSX:      &JSXParser{},
}

type PageParser interface {
	Parse(p *Page) ([]byte, Data, error)
}

func SplitFrontmatter(src []byte) (Data, []byte, error) {
	if !bytes.Contains(src, []byte("---\n")) {
		return nil, src, nil
	}
	parts := bytes.SplitN(src, []byte("---\n"), 3)
	if len(parts) != 3 {
		return nil, nil, fmt.Errorf("invalid frontmatter: %d", len(parts))
	}

	var data Data
	if err := yaml.Unmarshal(parts[1], &data); err != nil {
		return nil, nil, err
	}
	return data, parts[2], nil
}
