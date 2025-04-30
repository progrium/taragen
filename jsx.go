package taragen

import (
	"bytes"
	"fmt"
	"log"
	"slices"
	"strings"

	"github.com/dop251/goja"
	"github.com/evanw/esbuild/pkg/api"
)

func setupJSX(name string, src []byte) (*goja.Runtime, string, error) {
	transform := api.Transform(string(src), api.TransformOptions{
		Loader:         api.LoaderJSX,
		JSXFactory:     "hyper",
		JSXFragment:    "'<>'",
		JSXSideEffects: true,
	})
	if len(transform.Errors) > 0 {
		return nil, "", fmt.Errorf("error parsing JSX: %s (%s:%d)", transform.Errors[0].Text, name, transform.Errors[0].Location.Line)
	}

	vm := goja.New()
	if err := vm.Set("hyper", hyper); err != nil {
		return nil, "", err
	}

	return vm, string(transform.Code), nil
}

func RenderJSX(name string, src []byte, globals map[string]any, args ...any) ([]byte, error) {
	vm, jsCode, err := setupJSX(name, replacePreNewlines(src))
	if err != nil {
		return nil, err
	}

	// if there is a page global, we need to allow updating it with a data function
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

	v, err := vm.RunString(jsCode)
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

	var node hyperNode
	err = vm.ExportTo(v, &node)
	if err != nil {
		return nil, err
	}
	return []byte(replaceNewlines(node.String())), nil
}

func ExportJSX(name string, src []byte, globals map[string]any) (map[string]any, error) {
	vm, jsCode, err := setupJSX(name, src)
	if err != nil {
		return nil, err
	}

	out := map[string]any{}
	before := vm.GlobalObject().Keys()

	for key, value := range globals {
		if err := vm.Set(key, value); err != nil {
			return nil, err
		}
	}

	_, err = vm.RunString(jsCode)
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

func hyper(tag string, attrs map[string]any, children ...any) hyperNode {
	var nodes []hyperNode
	for _, child := range children {
		if child == nil {
			continue
		}
		switch c := child.(type) {
		case []any:
			nodes = append(nodes, hyper("", nil, c...).children...)
		case hyperNode:
			nodes = append(nodes, c)
		case string:
			nodes = append(nodes, hyperNode{text: c})
		case int64:
			nodes = append(nodes, hyperNode{text: fmt.Sprintf("%d", c)})
		default:
			log.Panicf("unsupported child type: %T (in %s)", child, tag)
		}
	}
	return hyperNode{tag: tag, attrs: attrs, children: nodes}
}

type hyperNode struct {
	tag      string
	attrs    map[string]any
	children []hyperNode
	text     string
}

func (h hyperNode) isSelfClosing() bool {
	return len(h.children) == 0 && slices.Contains([]string{
		"area",
		"base",
		"br",
		"col",
		"embed",
		"hr",
		"img",
		"input",
		"link",
		"meta",
		"param",
		"source",
		"track",
		"wbr",
	}, h.tag)
}

func (h hyperNode) String() string {
	if h.text != "" {
		return h.text
	}

	var builder strings.Builder

	if h.tag != "<>" {
		builder.WriteString("<" + h.tag)

		if len(h.attrs) > 0 {
			builder.WriteString(" ")
			var i int
			for k, v := range h.attrs {
				i++
				switch vv := v.(type) {
				case string:
					if vv == "" {
						continue
					}
					builder.WriteString(k + "=\"" + vv + "\"")
				case map[string]any:
					// assuming this is a style object
					var style strings.Builder
					for k, v := range vv {
						style.WriteString(k + ":" + v.(string) + ";")
					}
					builder.WriteString(k + "=\"" + style.String() + "\"")
				case bool:
					if vv {
						builder.WriteString(k)
					}
				default:
					log.Panicf("unsupported attr type: %T (in %s)", vv, h.tag)
				}

				if i < len(h.attrs) {
					builder.WriteString(" ")
				}
			}
		}

		if h.isSelfClosing() {
			builder.WriteString("/")
		}
		builder.WriteString(">")
	}

	for _, child := range h.children {
		builder.WriteString(child.String())
	}

	if !h.isSelfClosing() && h.tag != "<>" {
		builder.WriteString("</" + h.tag + ">")
	}
	return builder.String()
}

func replacePreNewlines(data []byte) []byte {
	// Replace newlines with <newline /> between <pre> tags
	parts := bytes.Split(data, []byte("<pre>"))
	for i := 1; i < len(parts); i++ {
		if preEnd := bytes.Index(parts[i], []byte("</pre>")); preEnd != -1 {
			preContent := parts[i][:preEnd]
			afterPre := parts[i][preEnd:]
			preContent = bytes.ReplaceAll(preContent, []byte("\n"), []byte("<newline />"))
			parts[i] = append(preContent, afterPre...)
		}
	}
	return bytes.Join(parts, []byte("<pre>"))
}

func replaceNewlines(s string) string {
	return strings.ReplaceAll(s, "<newline></newline>", "\n")
}
