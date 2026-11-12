package taragen

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/yosssi/gohtml"
)

type Data map[string]any

const (
	Layout      = "layout"
	Slug        = "slug"
	Path        = "path"
	Source      = "src"
	Body        = "body"
	IsDir       = "isDir"
	Date        = "date"
	ContentType = "contentType"
	Draft       = "draft"
	Preamble    = "preamble"
	Outputs     = "outputs"
)

type Page struct {
	path    string // normalized path to the page (ex: path/to/page)
	file    string // source file relative to the site root, once parsed
	data    Data
	site    *Site
	globals map[string]any
	outputs []*Output // outputs that apply to the page and are turned on
	aliases []string  // normalized aliases, as of the last parse

	// set while parsing: the output templates that apply, and the outputs
	// opt-out value read from page data
	templates []*Output
	optOut    any

	// set for generated pages: the template source relative to the site
	// root, and the parameter set seeded as page.params
	template string
	params   map[string]any
	// virtualDir is a bare directory added for a generated page's parent,
	// with no directory on disk
	virtualDir bool

	// usesPages is set when pages() is called while the page parses, so
	// ParseAll can parse it again once every page exists
	usesPages bool
	// links and backlinks are the page's place in the link graph, set by
	// the link pass at the end of ParseAll and kept across re-parses
	links     []any
	backlinks []any
}

// Template returns the template source a generated page renders from,
// relative to the site root, or "" for any other page.
func (p *Page) Template() string {
	return p.template
}

// URL returns the page's URL path, such as "/" or "/docs/intro".
func (p *Page) URL() string {
	return path.Clean("/" + p.path)
}

// Outputs returns the page's outputs that apply to it and are turned on, as
// of its last parse.
func (p *Page) Outputs() []*Output {
	return p.outputs
}

func getData[T any](data Data, key string, fallback T) T {
	if data[key] == nil {
		return fallback
	}
	return data[key].(T)
}

func (p *Page) Body() []byte {
	return []byte(getData(p.data, Body, ""))
}

func (p *Page) Source() []byte {
	return []byte(getData(p.data, Source, ""))
}

func (p *Page) IsDir() bool {
	return getData(p.data, IsDir, false)
}

func (p *Page) Draft() bool {
	if p.Date() != "" && p.Time().After(time.Now()) {
		return true
	}
	return getData(p.data, Draft, false)
}

func (p *Page) Date() string {
	return getData(p.data, Date, "")
}

func (p *Page) Time() time.Time {
	t, _ := time.Parse("2006-01-02", p.Date())
	return t
}

func (p *Page) Slug() string {
	return getData(p.data, Slug, "")
}

func (p *Page) ContentType() string {
	return getData(p.data, ContentType, "text/html")
}

func (p *Page) Preamble() string {
	preamble := getData(p.data, Preamble, "")
	if preamble == "" && p.ContentType() == "text/html" {
		preamble = "<!DOCTYPE html>"
	}
	return preamble
}

func (p *Page) Subpages() (subpages []*Page) {
	for _, page := range p.site.pages {
		if page == p {
			continue // the root is in its own directory
		}
		if strings.HasPrefix(page.path, p.path+"/") || p.path == "." {
			relPath := strings.TrimPrefix(page.path, p.path+"/")
			if strings.Contains(relPath, "/") {
				continue
			}
			if !p.site.drafts && page.Draft() {
				continue
			}
			subpages = append(subpages, page)
		}
	}
	sortPages(subpages)
	return subpages
}

func (p *Page) loadGlobals() error {
	// measureText is set before _globals.jsx runs, so helpers defined there
	// (like a word wrapper) can call it.
	globals := map[string]any{"measureText": measureTextFunc(p.site)}

	// find all _globals.jsx files in parent directories
	var globalsFiles []string
	dir := filepath.Dir(p.path)
	stop := false
	for {
		globalsPath := path.Join(dir, "_globals"+ExtJSX)
		if _, err := os.Stat(path.Join(p.site.dir, globalsPath)); err == nil {
			globalsFiles = append([]string{globalsPath}, globalsFiles...)
		}
		if stop {
			break
		}
		dir = filepath.Dir(dir)
		if dir == "." || dir == "/" {
			stop = true
		}
	}
	for _, globalsPath := range globalsFiles {
		globalsSrc, err := os.ReadFile(path.Join(p.site.dir, globalsPath))
		if err != nil {
			return err
		}
		g, err := ExportJSX(globalsPath, globalsSrc, globals)
		if err != nil {
			return err
		}
		for k, v := range g {
			globals[k] = v
		}
	}

	// load built-in globals
	for k, v := range builtinGlobals(p) {
		globals[k] = v
	}

	p.globals = globals
	return nil
}

func (p *Page) loadDefaults() error {
	// find all _defaults.jsx files in parent directories
	var dataFiles []string
	dir := filepath.Dir(p.path)
	stop := false
	for {
		dataPath := path.Join(dir, "_defaults"+ExtJSX)
		if _, err := os.Stat(path.Join(p.site.dir, dataPath)); err == nil {
			dataFiles = append([]string{dataPath}, dataFiles...)
		}
		if stop {
			break
		}
		dir = filepath.Dir(dir)
		if dir == "." || dir == "/" {
			stop = true
		}
	}
	for _, dataPath := range dataFiles {
		dataSrc, err := os.ReadFile(path.Join(p.site.dir, dataPath))
		if err != nil {
			return err
		}
		data, err := ExportJSX(dataPath, dataSrc, p.globals)
		if err != nil {
			return err
		}
		for k, v := range data {
			p.data[k] = v
		}
	}
	return nil
}

func (p *Page) Parse() error {
	var format string
	var src []byte
	var err error

	var suffixes []string
	for ext := range Formats {
		suffixes = append(suffixes, ext)
		suffixes = append(suffixes, "/index"+ext)
	}

	// page.outputs is always recomputed, so a value left by a previous parse
	// isn't read as an opt-out
	p.data[Outputs] = map[string]any{}
	p.outputs, p.templates, p.optOut = nil, nil, nil
	p.aliases = nil
	p.usesPages = false

	if p.virtualDir {
		return nil
	}
	if p.template != "" {
		// a generated page renders from its template, with its parameter
		// set always seeded
		p.data[Params] = p.params
		src, err = os.ReadFile(filepath.Join(p.site.dir, filepath.FromSlash(p.template)))
		if err != nil {
			return err
		}
		format = ExtJSX
		p.file = p.template
		if path.Base(p.template) == "index"+ExtJSX {
			p.data[IsDir] = true
		}
	}

	for _, suffix := range suffixes {
		if src != nil {
			break
		}
		src, err = os.ReadFile(path.Join(p.site.dir, p.path+suffix))
		if err == nil {
			format = filepath.Ext(suffix)
			p.file = p.path + suffix
			if strings.HasPrefix(suffix, "/index") {
				p.data[IsDir] = true
			}
			break
		}
	}

	if src == nil {
		fi, err := os.Stat(path.Join(p.site.dir, p.path))
		if err == nil && fi.IsDir() {
			p.data[IsDir] = true
			return nil
		}
		return fmt.Errorf("unable to find page source for: %s", p.path)
	}

	if err := p.loadGlobals(); err != nil {
		return err
	}
	if err := p.loadDefaults(); err != nil {
		return err
	}

	p.data[Source] = string(src)

	f, ok := Formats[format]
	if !ok {
		return fmt.Errorf("unknown page format: %s", format)
	}

	// page.outputs is set before the body runs, so the page can read it.
	// Opt-outs set by defaults apply now, and the page's own opt-outs replace
	// them as they are read.
	if p.templates, err = p.outputTemplates(); err != nil {
		return err
	}
	p.setOptOut(p.data[Outputs])
	p.setGraphData()

	body, data, err := f.Parse(p)
	if err != nil {
		return err
	}
	optOut, setByFrontmatter := data[Outputs]
	for k, v := range data {
		p.data[k] = v
	}
	p.data[Body] = string(body)
	if setByFrontmatter {
		p.optOut = optOut
	}
	if err := p.normalizeAliases(); err != nil {
		return err
	}

	if p.ContentType() == "text/html" {
		p.outputs = enabledOutputs(p.templates, p.optOut)
	}
	p.data[Outputs] = outputsMap(p.outputs)
	p.setGraphData()

	return nil
}

// setOptOut records an outputs opt-out value and sets page.outputs to the
// outputs it leaves on.
func (p *Page) setOptOut(optOut any) {
	p.optOut = optOut
	p.data[Outputs] = outputsMap(enabledOutputs(p.templates, optOut))
}

// onData is called after each data() call in a JSX page body, so a page
// that turns outputs off never reads its own opt-out back from page.outputs.
func (p *Page) onData(data map[string]any) {
	if v, ok := data[Outputs]; ok {
		p.setOptOut(v)
	}
}

// enabledOutputs filters templates by an outputs opt-out value.
func enabledOutputs(templates []*Output, optOut any) []*Output {
	var outputs []*Output
	for _, o := range templates {
		if outputEnabled(optOut, o.Name) {
			outputs = append(outputs, o)
		}
	}
	return outputs
}

func (p *Page) Render(w io.Writer) (err error) {
	if err = p.Parse(); err != nil {
		return
	}

	origLayout := p.data[Layout]
	defer func() {
		p.data[Layout] = origLayout
	}()

	// Find all layout files in parent directories
	defaultLayouts := []string{}
	if origLayout == nil {
		dir := p.path
		for {
			// Check for JSX layout
			jsxLayoutPath := path.Join(dir, "_layout"+ExtJSX)
			if _, err := os.Stat(path.Join(p.site.dir, jsxLayoutPath)); err == nil {
				defaultLayouts = append(defaultLayouts, strings.TrimSuffix(jsxLayoutPath, ExtJSX))
			}
			// Check for Template layout
			tplLayoutPath := path.Join(dir, "_layout"+ExtTemplate)
			if _, err := os.Stat(path.Join(p.site.dir, tplLayoutPath)); err == nil {
				defaultLayouts = append(defaultLayouts, strings.TrimSuffix(tplLayoutPath, ExtTemplate))
			}

			if dir == "." || dir == "/" {
				break
			}
			dir = filepath.Dir(dir)
		}

		if len(defaultLayouts) > 0 && p.data[Layout] == nil {
			p.data[Layout] = defaultLayouts[0]
			defaultLayouts = defaultLayouts[1:]
		}
	}

	out := p.Body()
	for {
		layout, ok := p.data[Layout].(string)
		if !ok || layout == "" {
			break
		}
		// TODO: could this be replaced with call to p.Site.Partial?
		layoutPath := path.Join(p.site.dir, layout)
		isJSX := true
		layoutSrc, err := os.ReadFile(layoutPath + ExtJSX)
		if err != nil {
			layoutSrc, err = os.ReadFile(layoutPath + ExtTemplate)
			if err != nil {
				return err
			}
			isJSX = false
		}
		if isJSX {
			// the content is rendered HTML, so the layout inserts it unescaped
			out, err = RenderJSX(layoutPath, layoutSrc, p.globals, rawNode(string(out)))
			if err != nil {
				return err
			}
		} else {
			data, rest, err := SplitFrontmatter(layoutSrc)
			if err != nil {
				return err
			}
			for k, v := range data {
				p.data[k] = v
			}
			out, err = RenderTemplate(layoutPath, rest, builtinFuncs(p, out))
			if err != nil {
				return err
			}
		}

		if p.data[Layout].(string) == layout {
			if len(defaultLayouts) > 0 {
				p.data[Layout] = defaultLayouts[0]
				defaultLayouts = defaultLayouts[1:]
				continue
			}
			break
		} else {
			defaultLayouts = nil
		}
	}
	_, err = w.Write([]byte(p.Preamble() + "\n"))
	if err != nil {
		return
	}
	if p.ContentType() == "text/html" {
		gohtml.Condense = true
		out = gohtml.FormatBytes(out)
	}
	_, err = w.Write(out)
	return
}
