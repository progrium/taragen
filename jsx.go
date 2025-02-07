package taragen

import (
	"fmt"
	"log"
	"slices"
	"strings"

	"github.com/dop251/goja"
	"github.com/evanw/esbuild/pkg/api"
)

type JSXFormat struct{}

func (f *JSXFormat) Parse(p *Page) (out []byte, data Data, err error) {
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
	out, err = RenderJSX(p.Source(), p.jsxGlobals())
	return
}

func RenderJSX(src []byte, globals map[string]any, args ...any) ([]byte, error) {
	transform := api.Transform(string(src), api.TransformOptions{
		Loader:         api.LoaderJSX,
		JSXFactory:     "hyper",
		JSXFragment:    "'<>'",
		JSXSideEffects: true,
	})
	if len(transform.Errors) > 0 {
		return nil, fmt.Errorf("error parsing module: %s", transform.Errors[0].Text)
	}

	vm := goja.New()
	vm.SetFieldNameMapper(goja.UncapFieldNameMapper())

	if err := vm.Set("hyper", hyper); err != nil {
		return nil, err
	}

	// if there is a page global, we need to update it with a data function
	page, ok := globals["page"].(Data)
	if ok {
		if err := vm.Set("data", func(call goja.FunctionCall) goja.Value {
			var data map[string]any
			vm.ExportTo(call.Argument(0), &data)
			for k, v := range data {
				vm.Get("page").ToObject(vm).Set(k, v)
				page[k] = v
			}
			return nil
		}); err != nil {
			return nil, err
		}
	}

	for key, value := range globals {
		if err := vm.Set(key, value); err != nil {
			return nil, err
		}
	}

	v, err := vm.RunString(string(transform.Code))
	if err != nil {
		return nil, err
	}

	fn, ok := goja.AssertFunction(v)
	if ok {
		var jsArgs []goja.Value
		for _, arg := range args {
			jsArgs = append(jsArgs, vm.ToValue(arg))
		}
		v, err = fn(goja.Undefined(), jsArgs...)
		if err != nil {
			return nil, err
		}
	}

	var node HyperNode
	err = vm.ExportTo(v, &node)
	if err != nil {
		return nil, err
	}
	return []byte(node.String()), nil
}

func ExportJSX(src []byte, globals map[string]any) (map[string]any, error) {
	transform := api.Transform(string(src), api.TransformOptions{
		Loader:         api.LoaderJSX,
		JSXFactory:     "hyper",
		JSXFragment:    "'<>'",
		JSXSideEffects: true,
	})
	if len(transform.Errors) > 0 {
		return nil, fmt.Errorf("error parsing module: %s", transform.Errors[0].Text)
	}

	vm := goja.New()

	if err := vm.Set("hyper", hyper); err != nil {
		return nil, err
	}

	out := map[string]any{}
	before := vm.GlobalObject().Keys()

	for key, value := range globals {
		if err := vm.Set(key, value); err != nil {
			return nil, err
		}
	}

	_, err := vm.RunString(string(transform.Code))
	if err != nil {
		return nil, err
	}

	for _, key := range vm.GlobalObject().Keys() {
		if !slices.Contains(before, key) {
			out[key] = vm.Get(key).Export()
		}
	}

	return out, nil
}

type HyperNode struct {
	Tag      string
	Attrs    map[string]string
	Children []HyperNode
	Text     string
}

func (h HyperNode) isSelfClosing() bool {
	return len(h.Children) == 0 && !slices.Contains([]string{
		"script",
		"link",
		"iframe",
	}, h.Tag)
}

func (h HyperNode) String() string {
	if h.Text != "" {
		return h.Text
	}

	var builder strings.Builder

	if h.Tag == "cdata" {
		builder.WriteString("<![CDATA[")

		for _, child := range h.Children {
			builder.WriteString(child.String())
		}

		builder.WriteString("]]>")
		return builder.String()
	}

	if h.Tag != "<>" {
		if h.Tag == "xml" {
			builder.WriteString("<?" + h.Tag)
		} else {
			builder.WriteString("<" + h.Tag)
		}

		if len(h.Attrs) > 0 {
			builder.WriteString(" ")
			var i int
			for k, v := range h.Attrs {
				i++
				builder.WriteString(k + "=\"" + v + "\"")
				if i < len(h.Attrs) {
					builder.WriteString(" ")
				}
			}
		}

		if h.Tag == "xml" {
			builder.WriteString("?")
		} else if h.isSelfClosing() {
			builder.WriteString("/")
		}
		builder.WriteString(">")
	}

	for _, child := range h.Children {
		builder.WriteString(child.String())
	}

	if !h.isSelfClosing() && h.Tag != "<>" && h.Tag != "xml" {
		builder.WriteString("</" + h.Tag + ">")
	}
	return builder.String()
}

func hyper(tag string, attrs map[string]string, children ...any) HyperNode {
	var nodes []HyperNode
	for _, child := range children {
		if child == nil {
			continue
		}
		switch c := child.(type) {
		case []any:
			nodes = append(nodes, hyper("", nil, c...).Children...)
		case HyperNode:
			nodes = append(nodes, c)
		case string:
			nodes = append(nodes, HyperNode{Text: c})
		default:
			// nodes = append(nodes, HyperNode{Text: " "})
			log.Panicf("unsupported child type: %T (in %s)", child, tag)
		}
	}
	return HyperNode{Tag: tag, Attrs: attrs, Children: nodes}
}
