package taragen

import (
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
	out, err = RenderJSX(p.Source(), p.globals)
	return
}
