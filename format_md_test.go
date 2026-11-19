package taragen

import (
	"strings"
	"testing"
)

func renderMarkdown(t *testing.T, src string) string {
	t.Helper()
	dir := writeSite(t, map[string]string{"page.md": src})
	return mustRender(t, dir, "page")
}

func TestMarkdownHighlightClasses(t *testing.T) {
	out := renderMarkdown(t, "```go\nfunc main() {}\n```\n")
	assertContains(t, out,
		`<pre tabindex="0" class="chroma"><code class="language-go">`,
		`<span class="line"><span class="cl">`,
		`<span class="kd">func</span>`,
		`<span class="nf">main</span>`,
		`</code></pre>`,
	)
	if strings.Contains(out, "style=") {
		t.Errorf("highlighted output has a style attribute:\n%s", out)
	}
}

func TestMarkdownHighlightUnknownLanguage(t *testing.T) {
	out := renderMarkdown(t, "```nosuchlang\na < b\n```\n\n```\nplain\n```\n\n    indented\n")
	assertContains(t, out,
		"<pre><code class=\"language-nosuchlang\">a &lt; b\n</code></pre>",
		"<pre><code>plain\n</code></pre>",
		"<pre><code>indented\n</code></pre>",
	)
	if strings.Contains(out, "chroma") || strings.Contains(out, "style=") {
		t.Errorf("unhighlighted blocks have highlighting markup:\n%s", out)
	}
}

func TestMarkdownHighlightEscapes(t *testing.T) {
	out := renderMarkdown(t, "```go\nx := \"<a&b>\"\n```\n")
	assertContains(t, out, `<span class="s">&#34;&lt;a&amp;b&gt;&#34;</span>`)
	if strings.Contains(out, "<a&b>") {
		t.Errorf("token text is not escaped:\n%s", out)
	}
}

func TestMarkdownGFM(t *testing.T) {
	src := "| Name | Count |\n| :--- | ----: |\n| a | 1 |\n\n" +
		"~~gone~~\n\n" +
		"- [x] done\n- [ ] todo\n\n" +
		"see https://example.com\n"
	out := renderMarkdown(t, src)
	assertContains(t, out,
		"<table>",
		`<th style="text-align:left">Name</th>`,
		`<td style="text-align:right">1</td>`,
		"<del>gone</del>",
		`<input checked="" disabled="" type="checkbox">`,
		`<input disabled="" type="checkbox">`,
		`<a href="https://example.com">https://example.com</a>`,
	)
}
