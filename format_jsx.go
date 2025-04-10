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
				return runtime.ToValue(err.Error())
			}
			return runtime.ToValue(string(partial))
		},
		"pages": func(call goja.FunctionCall, runtime *goja.Runtime) goja.Value {
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
			pages := p.site.Pages(call.Argument(0).String())
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
	var src []byte
	if !hasClosingTag(p.Source()) {
		src = []byte("<>")
		src = append(src, p.Source()...)
		src = append(src, []byte("</>")...)
	} else {
		src = p.Source()
	}
	out, err = RenderJSX(p.path, src, p.globals)
	return
}

func hasClosingTag(data []byte) bool {
	// Handle empty input
	if len(data) == 0 {
		return false
	}

	// Trim trailing newlines to handle the case where data ends with newline(s)
	trimmedData := bytes.TrimRight(data, "\n\r")
	if len(trimmedData) == 0 {
		return false // All data was newlines
	}

	// Find the last line by locating the last newline
	lastLineIndex := bytes.LastIndex(trimmedData, []byte("\n"))

	// If no newline found, check the entire data
	var lastLine []byte
	if lastLineIndex == -1 {
		lastLine = trimmedData
	} else {
		// Get the last line (exclude the newline character)
		lastLine = trimmedData[lastLineIndex+1:]
	}

	// Trim leading whitespace
	lastLine = bytes.TrimLeftFunc(lastLine, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\r'
	})

	// Check if it starts with "</"
	return bytes.HasPrefix(lastLine, []byte("</"))
}
