package taragen

import (
	"bytes"
	"fmt"
	"html"

	chromahtml "github.com/alecthomas/chroma/formatters/html"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting"
	gmhtml "github.com/yuin/goldmark/renderer/html"
)

type MarkdownParser struct{}

// highlighter renders highlighted fenced code blocks with Chroma's CSS class
// names instead of inline styles, so the site's stylesheet decides how code
// looks (see the highlight-classes proposal).
var highlighter = highlighting.NewHighlighting(
	highlighting.WithFormatOptions(chromahtml.WithClasses(true)),
	highlighting.WithCodeBlockOptions(func(c highlighting.CodeBlockContext) []chromahtml.Option {
		lang, _ := c.Language()
		return []chromahtml.Option{chromahtml.WithPreWrapper(preWrapper{lang: string(lang)})}
	}),
)

// preWrapper writes the <pre> and <code> around a highlighted block, keeping
// the block's language as a language-<lang> class on <code>.
type preWrapper struct{ lang string }

func (p preWrapper) Start(code bool, _ string) string {
	if !code {
		return `<pre tabindex="0" class="chroma">`
	}
	return fmt.Sprintf(`<pre tabindex="0" class="chroma"><code class="language-%s">`, html.EscapeString(p.lang))
}

func (p preWrapper) End(code bool) string {
	if !code {
		return "</pre>"
	}
	return "</code></pre>"
}

func (f *MarkdownParser) Parse(p *Page) ([]byte, Data, error) {
	src, data, err := new(TemplateParser).Parse(p)
	if err != nil {
		return nil, nil, err
	}

	md := goldmark.New(
		goldmark.WithExtensions(highlighter),
		goldmark.WithRendererOptions(
			gmhtml.WithUnsafe(),
		),
	)
	var buf bytes.Buffer
	if err := md.Convert(src, &buf); err != nil {
		return nil, nil, err
	}

	return buf.Bytes(), data, nil
}
