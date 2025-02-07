package taragen

import (
	"bytes"

	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting"
	"github.com/yuin/goldmark/renderer/html"
)

type MarkdownParser struct{}

func (f *MarkdownParser) Parse(p *Page) ([]byte, Data, error) {
	src, data, err := new(TemplateParser).Parse(p)
	if err != nil {
		return nil, nil, err
	}

	md := goldmark.New(
		goldmark.WithExtensions(highlighting.Highlighting),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
	)
	var buf bytes.Buffer
	if err := md.Convert(src, &buf); err != nil {
		return nil, nil, err
	}

	return buf.Bytes(), data, nil
}
