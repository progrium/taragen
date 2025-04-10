package taragen

import (
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
