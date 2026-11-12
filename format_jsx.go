package taragen

import (
	"bytes"
	"fmt"
	"log"

	"github.com/dop251/goja"
)

type JSXParser struct{}

func builtinGlobals(p *Page) map[string]any {
	return map[string]any{
		"page": p.data,
		"partial": func(call goja.FunctionCall, runtime *goja.Runtime) goja.Value {
			name := call.Argument(0).String()
			var args []any
			for _, arg := range call.Arguments[1:] {
				args = append(args, arg.Export())
			}
			partial, err := p.site.Partial(name, p.globals, args...)
			if err != nil {
				// throw, so a broken partial fails the render instead of
				// rendering its error message into the page
				panic(runtime.NewGoError(fmt.Errorf("partial %s: %w", name, err)))
			}
			// partial output is already HTML, so it is inserted unescaped
			return runtime.ToValue(rawNode(string(partial)))
		},
		"pages": func(call goja.FunctionCall, runtime *goja.Runtime) goja.Value {
			// a body that reads other pages can change once every page is
			// parsed, so ParseAll parses it again
			p.usesPages = true
			// Helper function to build page data with recursive subpages
			var buildPageData func(*Page) map[string]any
			buildPageData = func(page *Page) map[string]any {
				pageData := make(map[string]any)
				// Copy all page data
				for k, v := range page.data {
					pageData[k] = v
				}

				// Get subpages
				var subpagesData []map[string]any
				for _, subpage := range page.Subpages() {
					if subpage != page { // Avoid self-reference
						subpagesData = append(subpagesData, buildPageData(subpage))
					}
				}
				pageData["subpages"] = subpagesData

				return pageData
			}

			// Convert pages to the new structure
			pages, err := queryPages(p.site, call.Argument(0))
			if err != nil {
				panic(runtime.NewGoError(err))
			}
			result := make([]map[string]any, len(pages))
			for i, page := range pages {
				result[i] = buildPageData(page)
			}

			return runtime.ToValue(result)
		},
	}
}

func (f *JSXParser) Parse(p *Page) (out []byte, data Data, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("panic on page:", p.path)
			switch panicErr := r.(type) {
			case error:
				err = panicErr
			default:
				err = fmt.Errorf("%v", panicErr)
			}
		}
	}()
	// wrapping in fragment to allow for multiple root elements
	// but also ensure partial and other jsx expressions render
	src, err := wrapJSX(p.Source())
	if err != nil {
		return nil, Data{}, err
	}
	out, err = renderJSX(p.path, src, p.globals, p.onData)
	if err != nil {
		fmt.Printf("error source: %s: %s\n", p.path, string(src))
		return
	}
	return
}

// wrapJSX wraps a page source in a fragment so it can have several root
// elements. A leading `data({...})` statement is kept outside the fragment. It
// is found by scanning the source, not by searching for the word "data", so
// pages that only mention data in their content are wrapped whole.
func wrapJSX(src []byte) ([]byte, error) {
	start := skipSpaceAndComments(src, 0)
	if !bytes.HasPrefix(src[start:], []byte("data")) {
		return fragmentWrap(src), nil
	}
	open := skipSpaceAndComments(src, start+len("data"))
	if open >= len(src) || src[open] != '(' {
		return fragmentWrap(src), nil
	}
	end, err := matchParen(src, open)
	if err != nil {
		return nil, err
	}
	end++ // past ')'
	if next := skipSpaceAndComments(src, end); next < len(src) && src[next] == ';' {
		end = next + 1
	}
	// the semicolon is required here: without it, `data(...)\n<>` would
	// parse as a comparison
	out := append([]byte{}, src[:end]...)
	out = append(out, ";\n"...)
	out = append(out, fragmentWrap(bytes.TrimLeft(src[end:], " \t\r\n"))...)
	return out, nil
}

// skipSpaceAndComments returns the index of the first byte at or after i that
// isn't whitespace or part of a JS comment.
func skipSpaceAndComments(src []byte, i int) int {
	for i < len(src) {
		switch {
		case src[i] == ' ' || src[i] == '\t' || src[i] == '\n' || src[i] == '\r':
			i++
		case bytes.HasPrefix(src[i:], []byte("//")):
			nl := bytes.IndexByte(src[i:], '\n')
			if nl == -1 {
				return len(src)
			}
			i += nl + 1
		case bytes.HasPrefix(src[i:], []byte("/*")):
			end := bytes.Index(src[i+2:], []byte("*/"))
			if end == -1 {
				return len(src)
			}
			i += 2 + end + 2
		default:
			return i
		}
	}
	return i
}

// matchParen returns the index of the parenthesis closing the one at open,
// skipping strings, template literals, and comments.
func matchParen(src []byte, open int) (int, error) {
	depth := 0
	for i := open; i < len(src); i++ {
		c := src[i]
		switch {
		case c == '"' || c == '\'' || c == '`':
			j, err := skipString(src, i)
			if err != nil {
				return 0, err
			}
			i = j
		case bytes.HasPrefix(src[i:], []byte("//")) || bytes.HasPrefix(src[i:], []byte("/*")):
			i = skipSpaceAndComments(src, i) - 1
		case c == '(' || c == '{' || c == '[':
			depth++
		case c == ')' || c == '}' || c == ']':
			depth--
			if depth == 0 {
				if c != ')' {
					return 0, fmt.Errorf("unbalanced data(...) statement")
				}
				return i, nil
			}
		}
	}
	return 0, fmt.Errorf("unterminated data(...) statement")
}

// skipString returns the index of the quote closing the string at i. Template
// literal substitutions are skipped by brace matching.
func skipString(src []byte, i int) (int, error) {
	quote := src[i]
	for j := i + 1; j < len(src); j++ {
		switch {
		case src[j] == '\\':
			j++
		case src[j] == quote:
			return j, nil
		case quote == '`' && bytes.HasPrefix(src[j:], []byte("${")):
			depth := 0
			for k := j + 1; k < len(src); k++ {
				switch src[k] {
				case '"', '\'', '`':
					end, err := skipString(src, k)
					if err != nil {
						return 0, err
					}
					k = end
				case '{':
					depth++
				case '}':
					depth--
				}
				if depth == 0 {
					j = k
					break
				}
			}
		}
	}
	return 0, fmt.Errorf("unterminated string in data(...) statement")
}

func fragmentWrap(data []byte) (out []byte) {
	out = []byte("<>")
	out = append(out, data...)
	out = append(out, []byte("</>")...)
	return
}

// queryPages runs a pages() call: a string is a path or pattern, and an
// object is the object form with path and where. Errors name the query.
func queryPages(site *Site, arg goja.Value) ([]*Page, error) {
	obj, isObj := arg.(*goja.Object)
	if !isObj {
		pages, err := site.Pages(arg.String())
		if err != nil {
			return nil, fmt.Errorf("pages(%q): %w", arg.String(), err)
		}
		return pages, nil
	}
	exported := obj.Export()
	fail := func(err error) ([]*Page, error) {
		return nil, fmt.Errorf("pages(%s): %w", describeQuery(exported), err)
	}
	m, ok := exported.(map[string]any)
	if !ok || obj.ClassName() != "Object" {
		return fail(fmt.Errorf("query must be a string or an object"))
	}
	q, err := parsePageQuery(m)
	if err != nil {
		return fail(err)
	}
	pages, err := site.Query(q.paths, q.where)
	if err != nil {
		return fail(err)
	}
	return pages, nil
}
