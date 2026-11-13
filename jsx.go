package taragen

import (
	"bytes"
	"fmt"
	"html"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/dop251/goja"
	"github.com/evanw/esbuild/pkg/api"
)

// compiledJSX caches compiled JSX by name and source. Every render gets a
// fresh VM, but a build runs the same _globals.jsx, layouts, partials, and
// output templates for every page, so transforming and compiling each one
// once instead of on every render saves most of the work of starting a VM.
// A goja Program isn't tied to a runtime and can be run by many.
var compiledJSX struct {
	sync.Mutex
	programs map[jsxKey]*goja.Program
}

type jsxKey struct{ name, src string }

// maxCompiledJSX bounds the cache, which serve would otherwise grow with
// every edit. Past it the cache starts over.
const maxCompiledJSX = 1024

// compileJSX transforms JSX source to JS and compiles it, or returns the
// program from an earlier call with the same name and source.
func compileJSX(name string, src []byte) (*goja.Program, error) {
	key := jsxKey{name, string(src)}
	compiledJSX.Lock()
	prg, ok := compiledJSX.programs[key]
	compiledJSX.Unlock()
	if ok {
		return prg, nil
	}

	transform := api.Transform(key.src, api.TransformOptions{
		Loader:         api.LoaderJSX,
		JSXFactory:     "hyper",
		JSXFragment:    "'<>'",
		JSXSideEffects: true,
	})
	if len(transform.Errors) > 0 {
		return nil, fmt.Errorf("error parsing JSX: %s (%s:%d)", transform.Errors[0].Text, name, transform.Errors[0].Location.Line)
	}
	// compiled without a name, as RunString does, so errors read the same
	prg, err := goja.Compile("", string(transform.Code), false)
	if err != nil {
		return nil, err
	}

	compiledJSX.Lock()
	if compiledJSX.programs == nil || len(compiledJSX.programs) >= maxCompiledJSX {
		compiledJSX.programs = make(map[jsxKey]*goja.Program)
	}
	compiledJSX.programs[key] = prg
	compiledJSX.Unlock()
	return prg, nil
}

func setupJSX(name string, src []byte) (*goja.Runtime, *goja.Program, error) {
	prg, err := compileJSX(name, src)
	if err != nil {
		return nil, nil, err
	}

	vm := goja.New()
	if err := vm.Set("hyper", hyperFunc(vm)); err != nil {
		return nil, nil, err
	}
	if err := vm.Set("raw", func(v goja.Value) hyperNode {
		if goja.IsUndefined(v) || goja.IsNull(v) {
			return rawNode("")
		}
		return rawNode(v.String())
	}); err != nil {
		return nil, nil, err
	}

	return vm, prg, nil
}

func RenderJSX(name string, src []byte, globals map[string]any, args ...any) ([]byte, error) {
	return renderJSX(name, src, globals, nil, args...)
}

// renderJSX is RenderJSX with onData, if set, called after each data() call
// with the values it merged into the page.
func renderJSX(name string, src []byte, globals map[string]any, onData func(map[string]any), args ...any) ([]byte, error) {
	vm, prg, err := setupJSX(name, replacePreNewlines(src))
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
			if onData != nil {
				onData(data)
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

	v, err := vm.RunProgram(prg)
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

	nodes, err := toNodes(vm, v, "")
	if err != nil {
		return nil, err
	}
	var builder strings.Builder
	for _, node := range nodes {
		node.write(&builder, false)
	}
	return []byte(replaceNewlines(builder.String())), nil
}

func ExportJSX(name string, src []byte, globals map[string]any) (map[string]any, error) {
	vm, prg, err := setupJSX(name, src)
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

	_, err = vm.RunProgram(prg)
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

// hyperFunc returns the JSX element factory for a VM. It reads attributes from
// the JS object directly so they keep their source order, and reports bad
// values as JS TypeErrors so they surface as render errors instead of panics.
func hyperFunc(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		tag := call.Argument(0).String()
		node := hyperNode{tag: tag}

		if a := call.Argument(1); !goja.IsUndefined(a) && !goja.IsNull(a) {
			obj := a.ToObject(vm)
			for _, key := range obj.Keys() {
				at, ok, err := toAttr(vm, key, obj.Get(key))
				if err != nil {
					panic(vm.NewTypeError(fmt.Sprintf("%s (in %s)", err, tag)))
				}
				if ok {
					node.attrs = append(node.attrs, at)
				}
			}
		}

		for _, child := range call.Arguments[min(2, len(call.Arguments)):] {
			nodes, err := toNodes(vm, child, tag)
			if err != nil {
				panic(vm.NewTypeError(err.Error()))
			}
			node.children = append(node.children, nodes...)
		}
		return vm.ToValue(node)
	}
}

func toAttr(vm *goja.Runtime, key string, v goja.Value) (attr hyperAttr, ok bool, err error) {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return attr, false, nil
	}
	switch vv := v.Export().(type) {
	case string:
		if vv == "" {
			return attr, false, nil
		}
		return hyperAttr{name: key, value: vv}, true, nil
	case int64, float64:
		return hyperAttr{name: key, value: v.String()}, true, nil
	case bool:
		if !vv {
			return attr, false, nil
		}
		return hyperAttr{name: key, bare: true}, true, nil
	case map[string]any:
		// assuming this is a style object
		obj := v.ToObject(vm)
		var style strings.Builder
		for _, k := range obj.Keys() {
			sv := obj.Get(k)
			if goja.IsUndefined(sv) || goja.IsNull(sv) {
				continue
			}
			style.WriteString(k + ":" + sv.String() + ";")
		}
		return hyperAttr{name: key, value: style.String()}, true, nil
	default:
		return attr, false, fmt.Errorf("unsupported attr type for %s: %T", key, vv)
	}
}

// toNodes converts a JS value used as a child (or a render result) to nodes.
// Strings become escaped text, numbers are formatted the way JS prints them,
// and null, undefined, and booleans render nothing, so `{cond && <x/>}` works.
func toNodes(vm *goja.Runtime, v goja.Value, parent string) ([]hyperNode, error) {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return nil, nil
	}
	switch c := v.Export().(type) {
	case hyperNode:
		if c.tag == "<>" {
			return c.children, nil
		}
		return []hyperNode{c}, nil
	case string:
		return []hyperNode{{text: c}}, nil
	case int64, float64:
		return []hyperNode{{text: v.String()}}, nil
	case bool:
		return nil, nil
	}
	if obj, ok := v.(*goja.Object); ok && obj.ClassName() == "Array" {
		var nodes []hyperNode
		length := obj.Get("length").ToInteger()
		for i := int64(0); i < length; i++ {
			n, err := toNodes(vm, obj.Get(strconv.FormatInt(i, 10)), parent)
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, n...)
		}
		return nodes, nil
	}
	return nil, fmt.Errorf("unsupported child type: %T (in %s)", v.Export(), parent)
}

type hyperAttr struct {
	name  string
	value string
	bare  bool
}

type hyperNode struct {
	tag      string
	attrs    []hyperAttr
	children []hyperNode
	text     string
	raw      bool // text is written as-is instead of escaped
}

func rawNode(s string) hyperNode {
	return hyperNode{text: s, raw: true}
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

// rawTextElements hold text the browser doesn't decode entities in, so their
// text children are written unescaped.
var rawTextElements = []string{"script", "style"}

func (h hyperNode) String() string {
	var builder strings.Builder
	h.write(&builder, false)
	return builder.String()
}

func (h hyperNode) write(builder *strings.Builder, rawText bool) {
	if h.tag == "" {
		// replacePreNewlines also rewrites newlines inside JS strings in a
		// <pre>, where the marker ends up in text instead of an element
		text := strings.ReplaceAll(h.text, newlineMarker, "\n")
		if h.raw || rawText {
			builder.WriteString(text)
		} else {
			builder.WriteString(escapeText(text))
		}
		return
	}

	if h.tag != "<>" {
		builder.WriteString("<" + h.tag)
		for _, a := range h.attrs {
			builder.WriteString(" " + a.name)
			if !a.bare {
				builder.WriteString("=\"" + escapeAttr(a.value) + "\"")
			}
		}
		if h.isSelfClosing() {
			builder.WriteString("/")
		}
		builder.WriteString(">")
	}

	childRaw := rawText || slices.Contains(rawTextElements, strings.ToLower(h.tag))
	for _, child := range h.children {
		child.write(builder, childRaw)
	}

	if !h.isSelfClosing() && h.tag != "<>" {
		builder.WriteString("</" + h.tag + ">")
	}
}

var textEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")

func escapeText(s string) string {
	return textEscaper.Replace(s)
}

func escapeAttr(s string) string {
	return html.EscapeString(s)
}

const newlineMarker = "<newline />"

func replacePreNewlines(data []byte) []byte {
	// Replace newlines with <newline /> between <pre> tags
	parts := bytes.Split(data, []byte("<pre>"))
	for i := 1; i < len(parts); i++ {
		if preEnd := bytes.Index(parts[i], []byte("</pre>")); preEnd != -1 {
			preContent := parts[i][:preEnd]
			afterPre := parts[i][preEnd:]
			preContent = bytes.ReplaceAll(preContent, []byte("\n"), []byte(newlineMarker))
			parts[i] = append(preContent, afterPre...)
		}
	}
	return bytes.Join(parts, []byte("<pre>"))
}

func replaceNewlines(s string) string {
	return strings.ReplaceAll(s, "<newline></newline>", "\n")
}
