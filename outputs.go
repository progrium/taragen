package taragen

import (
	"fmt"
	"io"
	"mime"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/dop251/goja"
	"github.com/yosssi/gohtml"
)

// OutputsDir is the directory that holds output templates, in any directory
// of the site.
const OutputsDir = "_outputs"

var outputName = regexp.MustCompile(`^[a-z][a-z0-9]*$`)

// Output is an extra file written for a page by an _outputs template.
type Output struct {
	// Name is the output's name, such as "card" for _outputs/card.png.jsx.
	Name string
	// Suffix is appended to the page's URL, such as ".card.png".
	Suffix string
	// Template is the template's path relative to the site root.
	Template string
	// Page is the page the output belongs to.
	Page *Page
}

// URL returns the output's URL: the page's URL with the suffix appended,
// with "index" standing in for the root page.
func (o *Output) URL() string {
	base := o.Page.URL()
	if base == "/" {
		base = "/index"
	}
	return base + o.Suffix
}

// parseOutputTemplate returns the name and suffix for an output template's
// file name, or ok false for a file that isn't one.
func parseOutputTemplate(filename string) (name, suffix string, ok bool, err error) {
	if strings.HasPrefix(filename, ".") || !strings.HasSuffix(filename, ExtJSX) {
		return "", "", false, nil
	}
	base := strings.TrimSuffix(filename, ExtJSX)
	name, _, _ = strings.Cut(base, ".")
	if !outputName.MatchString(name) {
		return "", "", false, fmt.Errorf("invalid output template name %q: names are a lowercase letter followed by lowercase letters and digits", filename)
	}
	return name, "." + base, true, nil
}

// outputTemplates finds the output templates that apply to the page: those
// in _outputs/ directories from the page's own path up to the root, with the
// nearest template winning for each name.
func (p *Page) outputTemplates() ([]*Output, error) {
	found := make(map[string]*Output)
	dir := p.path
	for {
		outputsDir := path.Join(dir, OutputsDir)
		var entries []os.DirEntry
		if fi, err := os.Stat(filepath.Join(p.site.dir, outputsDir)); err == nil && fi.IsDir() {
			if entries, err = os.ReadDir(filepath.Join(p.site.dir, outputsDir)); err != nil {
				return nil, err
			}
		}
		inDir := make(map[string]string)
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name, suffix, ok, err := parseOutputTemplate(e.Name())
			if err != nil {
				return nil, fmt.Errorf("%s: %w", outputsDir, err)
			}
			if !ok {
				continue
			}
			if other, dup := inDir[name]; dup {
				return nil, fmt.Errorf("%s: %s and %s both define output %q", outputsDir, other, e.Name(), name)
			}
			inDir[name] = e.Name()
			if _, nearer := found[name]; nearer {
				continue
			}
			found[name] = &Output{
				Name:     name,
				Suffix:   suffix,
				Template: path.Join(outputsDir, e.Name()),
				Page:     p,
			}
		}
		if dir == "." || dir == "/" {
			break
		}
		dir = path.Dir(dir)
	}
	outputs := make([]*Output, 0, len(found))
	for _, o := range found {
		outputs = append(outputs, o)
	}
	sort.Slice(outputs, func(i, j int) bool {
		return outputs[i].Name < outputs[j].Name
	})
	return outputs, nil
}

// outputEnabled reads an `outputs` opt-out value from page data: false turns
// every output off, and a map turns off each name set to false.
func outputEnabled(optOut any, name string) bool {
	switch v := optOut.(type) {
	case bool:
		return v
	case map[string]any:
		on, isBool := v[name].(bool)
		return !isBool || on
	case map[any]any: // from YAML frontmatter
		on, isBool := v[name].(bool)
		return !isBool || on
	}
	return true
}

// outputsMap returns page.outputs: output name to URL.
func outputsMap(outputs []*Output) map[string]any {
	m := make(map[string]any, len(outputs))
	for _, o := range outputs {
		m[o.Name] = o.URL()
	}
	return m
}

// isHTML reports whether a content type is text/html, ignoring parameters.
func isHTML(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	return err == nil && mediaType == "text/html"
}

// defaultContentType is the content type for an output's suffix, from its
// last extension.
func (o *Output) defaultContentType() string {
	if t := mime.TypeByExtension(path.Ext(o.Suffix)); t != "" {
		return t
	}
	return "application/octet-stream"
}

// Render re-parses the output's page and renders the output to w. It
// returns the output's content type.
func (o *Output) Render(w io.Writer) (contentType string, err error) {
	p := o.Page
	if err = p.Parse(); err != nil {
		return "", err
	}
	src, err := os.ReadFile(filepath.Join(p.site.dir, o.Template))
	if err != nil {
		return "", err
	}
	data := Data{}
	out, err := renderOutputJSX(o.Template, src, p.globals, data)
	if err != nil {
		return "", fmt.Errorf("output %s of %s: %w", o.Name, p.URL(), err)
	}
	contentType = getData(data, ContentType, o.defaultContentType())
	if preamble := getData(data, Preamble, ""); preamble != "" {
		if _, err = io.WriteString(w, preamble+"\n"); err != nil {
			return "", err
		}
	}
	if isHTML(contentType) {
		gohtml.Condense = true
		out = gohtml.FormatBytes(out)
	}
	_, err = w.Write(out)
	return contentType, err
}

// renderOutputJSX runs an output template in a fresh VM with the page's
// globals. data() writes to the output's own data instead of the page's. A
// string result is returned verbatim, and an element is serialized to HTML.
func renderOutputJSX(name string, src []byte, globals map[string]any, data Data) (out []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("%v", r)
			}
		}
	}()
	vm, jsCode, err := setupJSX(name, replacePreNewlines(src))
	if err != nil {
		return nil, err
	}
	for key, value := range globals {
		if err := vm.Set(key, value); err != nil {
			return nil, err
		}
	}
	if err := vm.Set("data", func(call goja.FunctionCall) goja.Value {
		var d map[string]any
		vm.ExportTo(call.Argument(0), &d)
		for k, v := range d {
			data[k] = v
		}
		return nil
	}); err != nil {
		return nil, err
	}

	v, err := vm.RunString(jsCode)
	if err != nil {
		return nil, err
	}
	if fn, ok := goja.AssertFunction(v); ok {
		if v, err = fn(goja.Undefined()); err != nil {
			return nil, err
		}
	}

	if s, ok := v.Export().(string); ok {
		return []byte(strings.ReplaceAll(s, newlineMarker, "\n")), nil
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
