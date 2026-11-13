package taragen

import (
	"fmt"
	"strings"
	"testing"
)

func TestJSX(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		globals  map[string]any
		args     []any
		expected string
	}{
		{
			name:     "basic element",
			globals:  map[string]any{"foo": "bar"},
			input:    `<div>{foo}</div>`,
			expected: `<div>bar</div>`,
		},
		{
			name:     "nested elements",
			globals:  map[string]any{"foo": "bar"},
			input:    `<div><span>{foo}</span></div>`,
			expected: `<div><span>bar</span></div>`,
		},
		{
			name:     "attributes",
			globals:  map[string]any{"foo": "bar"},
			input:    `<div class={foo}>hello</div>`,
			expected: `<div class="bar">hello</div>`,
		},
		{
			name:     "self closing",
			input:    `<img src="test.jpg"/>`,
			expected: `<img src="test.jpg"/>`,
		},
		{
			name:     "fragment",
			input:    `<><div>1</div><div>2</div></>`,
			expected: `<div>1</div><div>2</div>`,
		},
		{
			name:     "text is escaped",
			globals:  map[string]any{"title": `Tips & <Tricks>`},
			input:    `<h1>{title}</h1>`,
			expected: `<h1>Tips &amp; &lt;Tricks&gt;</h1>`,
		},
		{
			name:     "attributes are escaped",
			globals:  map[string]any{"title": `Say "hi" & bye`},
			input:    `<meta content={title}/>`,
			expected: `<meta content="Say &#34;hi&#34; &amp; bye"/>`,
		},
		{
			name:     "raw is not escaped",
			globals:  map[string]any{"body": `<p>a &amp; b</p>`},
			input:    `<div>{raw(body)}</div>`,
			expected: `<div><p>a &amp; b</p></div>`,
		},
		{
			name:     "script and style text is not escaped",
			input:    "<><script>{`if (a < b && c) {}`}</script><style>{`a > b {}`}</style></>",
			expected: `<script>if (a < b && c) {}</script><style>a > b {}</style>`,
		},
		{
			name:     "attributes keep source order",
			input:    `<a href="/" class="x" id="y" title="z">a</a>`,
			expected: `<a href="/" class="x" id="y" title="z">a</a>`,
		},
		{
			name:     "style object keeps order",
			input:    `<div style={{color: "red", margin: 0, padding: "1px"}}></div>`,
			expected: `<div style="color:red;margin:0;padding:1px;"></div>`,
		},
		{
			name:     "numbers",
			input:    `<div width={24} data-x={1.5}>{1.5}{2}{0.1 + 0.2}</div>`,
			expected: `<div width="24" data-x="1.5">1.520.30000000000000004</div>`,
		},
		{
			name:     "booleans and null render nothing",
			input:    `<div>{false}{true}{null}{undefined}{false && <b/>}</div>`,
			expected: `<div></div>`,
		},
		{
			name:     "arrays of mixed children",
			input:    `<ul>{[1, "a", <li>b</li>, [2.5, null]]}</ul>`,
			expected: `<ul>1a<li>b</li>2.5</ul>`,
		},
		{
			name:     "newline marker in a pre string",
			input:    "<pre>{`a\nb`}</pre>",
			expected: "<pre>a\nb</pre>",
		},
		{
			name:     "function component",
			input:    `(name) => <div>Hello {name}</div>`,
			args:     []any{"World"},
			expected: `<div>Hello World</div>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RenderJSX(tt.name, []byte(tt.input), tt.globals, tt.args...)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(result) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, string(result))
			}
		})
	}
}

func TestJSXWithPageData(t *testing.T) {
	pageData := Data{"foo": "input", "bar": "input"}
	globals := map[string]any{"page": pageData}
	input := `data({bar: "page"}); <div>{page.foo}_{page.bar}</div>`

	result, err := RenderJSX("test", []byte(input), globals)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := `<div>input_page</div>`
	if string(result) != expected {
		t.Errorf("expected %q, got %q", expected, string(result))
	}

	if pageData["bar"] != "page" {
		t.Fatal("page data not updated")
	}
}

func TestJSXBadValuesAreErrors(t *testing.T) {
	for _, input := range []string{
		`<div>{{a: 1}}</div>`,
		`<div x={[1]}></div>`,
		`<div>{() => 1}</div>`,
	} {
		if _, err := RenderJSX("bad", []byte(input), nil); err == nil {
			t.Errorf("%s: expected an error", input)
		}
	}
}

func TestWrapJSX(t *testing.T) {
	page := func(src string) string {
		pageData := Data{}
		out, err := wrapJSX([]byte(src))
		if err != nil {
			t.Fatalf("%q: wrap: %v", src, err)
		}
		result, err := RenderJSX("test", out, map[string]any{"page": pageData})
		if err != nil {
			t.Fatalf("%q: render: %v", src, err)
		}
		return string(result) + "|" + fmt.Sprint(pageData["title"])
	}
	tests := []struct {
		input    string
		expected string
	}{
		{`<p>this page talks about data; and more</p>`, `<p>this page talks about data; and more</p>|<nil>`},
		{`<p>a</p><p>b</p>`, `<p>a</p><p>b</p>|<nil>`},
		{`data({title: "a; b"}); <p>{page.title}</p>`, `<p>a; b</p>|a; b`},
		{"data({title: `x ${\"(\"} y`})\n<p>{page.title}</p><p>2</p>", `<p>x ( y</p><p>2</p>|x ( y`},
		{"// set the title\n/* block */ data({\n  title: \"t\", // c;\n});\n<p>1</p><p>2</p>", `<p>1</p><p>2</p>|t`},
	}
	for _, tt := range tests {
		if got := page(tt.input); got != tt.expected {
			t.Errorf("%q: expected %q, got %q", tt.input, tt.expected, got)
		}
	}
	if _, err := wrapJSX([]byte(`data({title: "x"`)); err == nil {
		t.Error("expected an error for an unterminated data statement")
	}
}

func TestJSXValues(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`<input n={3} f={1.5} on={true} off={false} empty="" u={undefined} z={null}/>`, `<input n="3" f="1.5" on/>`},
		{`<p style={{color: "red", gone: null, also: undefined}}/>`, `<p style="color:red;"></p>`},
		{`<p>{1}{2.5}{true}{false}{null}{undefined}</p>`, `<p>12.5</p>`},
		{`<p>{[1, [2, "x"]]}</p>`, `<p>12x</p>`},
		{`<p>{raw(null)}{raw(undefined)}{raw("<b/>")}</p>`, `<p><b/></p>`},
		{`<p>{"a"}</p>`, `<p>a</p>`},
	}
	for _, tt := range tests {
		got, err := RenderJSX("test", []byte(tt.input), nil)
		if err != nil {
			t.Errorf("%s: %v", tt.input, err)
			continue
		}
		if string(got) != tt.expected {
			t.Errorf("%s: expected %q, got %q", tt.input, tt.expected, string(got))
		}
	}
}

func TestJSXErrors(t *testing.T) {
	for _, input := range []string{
		`<p>{</p>`,
		`throw new Error("boom")`,
		`() => { throw new Error("boom") }`,
		`<p>{[{a: 1}]}</p>`,
	} {
		if _, err := RenderJSX("bad", []byte(input), nil); err == nil {
			t.Errorf("%s: expected an error", input)
		}
	}
	if _, err := RenderJSX("bad", []byte(`<p/>`), map[string]any{"page": Data{}, "data": 1}); err != nil {
		t.Errorf("globals should be able to replace data: %v", err)
	}
	if _, err := ExportJSX("bad", []byte(`var x = ;`), nil); err == nil {
		t.Error("expected a syntax error from ExportJSX")
	}
}

func TestWrapJSXScanning(t *testing.T) {
	ok := []struct {
		input    string
		expected string
	}{
		// not a data call
		{`dataset`, `<>dataset</>`},
		{`data`, `<>data</>`},
		{`// only a comment`, `<>// only a comment</>`},
		{`/* unterminated`, `<>/* unterminated</>`},
		// strings, escapes, and comments inside the call
		{`data({a: 'x)\'y', b: "q\")"}) <p/>`, "data({a: 'x)\\'y', b: \"q\\\")\"});\n<><p/></>"},
		{"data({a: `${ {b: \")\"}.b }`}) <p/>", "data({a: `${ {b: \")\"}.b }`});\n<><p/></>"},
		{"data({a: 1, // )\n /* ) */ b: 2})\n<p/>", "data({a: 1, // )\n /* ) */ b: 2});\n<><p/></>"},
	}
	for _, tt := range ok {
		got, err := wrapJSX([]byte(tt.input))
		if err != nil {
			t.Errorf("%q: %v", tt.input, err)
			continue
		}
		if string(got) != tt.expected {
			t.Errorf("%q: expected %q, got %q", tt.input, tt.expected, string(got))
		}
	}
	for _, input := range []string{
		`data({a: 1]`,
		`data({a: "x})`,
		"data({a: `${\"x}`})",
	} {
		if _, err := wrapJSX([]byte(input)); err == nil {
			t.Errorf("%q: expected an error", input)
		}
	}
}

func TestCompileJSXCache(t *testing.T) {
	src := []byte(`<p>hi</p>`)
	a, err := compileJSX("a.jsx", src)
	if err != nil {
		t.Fatal(err)
	}
	b, err := compileJSX("a.jsx", src)
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Error("the same name and source compiled twice")
	}
	c, err := compileJSX("a.jsx", []byte(`<p>bye</p>`))
	if err != nil {
		t.Fatal(err)
	}
	if c == a {
		t.Error("changed source returned the cached program")
	}
	if _, err := compileJSX("bad.jsx", []byte(`<p>`)); err == nil || !strings.Contains(err.Error(), "bad.jsx") {
		t.Errorf("want a parse error naming the file, got %v", err)
	}
	// a cached program runs in a fresh VM each time, sharing no state
	for i := 0; i < 2; i++ {
		out, err := RenderJSX("count.jsx", []byte("var n = (typeof n === 'undefined' ? 0 : n) + 1;\n<p>{n}</p>"), nil)
		if err != nil {
			t.Fatal(err)
		}
		if string(out) != "<p>1</p>" {
			t.Errorf("render %d: got %s", i, out)
		}
	}
}
