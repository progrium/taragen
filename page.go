package taragen

import (
	"fmt"
	"html/template"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/yosssi/gohtml"
)

type Data map[string]any

const (
	Layout = "layout"

	Slug   = "slug"
	Path   = "path"
	Source = "src"
	Body   = "body"
	IsDir  = "isDir"
	Date   = "date"
)

type Page struct {
	path string // normalized path to the page (ex: path/to/page)
	data Data
	site *Site

	defaultGlobals map[string]any
}

func sortPages(pages []*Page) {
	sort.Slice(pages, func(i, j int) bool {
		dateI, dateJ := pages[i].Date(), pages[j].Date()
		// If date is not empty, parse and compare it; otherwise, compare by slug
		if dateI != "" && dateJ != "" {
			timeI, errI := time.Parse("2006-01-02", dateI)
			timeJ, errJ := time.Parse("2006-01-02", dateJ)
			if errI == nil && errJ == nil {
				return timeI.Before(timeJ)
			}
		}
		return pages[i].Slug() < pages[j].Slug()
	})
}

func getData[T any](data Data, key string) T {
	var empty T
	if data[key] == nil {
		return empty
	}
	return data[key].(T)
}

func (p *Page) Body() []byte {
	return []byte(getData[string](p.data, Body))
}

func (p *Page) Source() []byte {
	return []byte(getData[string](p.data, Source))
}

func (p *Page) IsDir() bool {
	return getData[bool](p.data, IsDir)
}

func (p *Page) Date() string {
	return getData[string](p.data, Date)
}

func (p *Page) Slug() string {
	return getData[string](p.data, Slug)
}

func (p *Page) Subpages() (subpages []*Page) {
	for _, page := range p.site.pages {
		if strings.HasPrefix(page.path, p.path+"/") {
			relPath := strings.TrimPrefix(page.path, p.path+"/")
			if strings.Contains(relPath, "/") {
				continue
			}
			subpages = append(subpages, page)
		}
	}
	sortPages(subpages)
	return subpages
}

func (p *Page) templateFuncs() template.FuncMap {
	return template.FuncMap{
		"content": func() template.HTML {
			return template.HTML(string(p.Body()))
		},
		"page": func(key string) string {
			return p.data[key].(string)
		},
		"partial": func(name string, args ...any) (template.HTML, error) {
			b, err := p.site.Partial(name, p.jsxGlobals(), args...)
			if err != nil {
				return "", err
			}
			return template.HTML(b), nil
		},
	}
}

func (p *Page) jsxGlobals() map[string]any {
	g := make(map[string]any)
	// TODO: load site globals
	for k, v := range p.defaultGlobals {
		g[k] = v
	}
	g["page"] = p.data
	g["partial"] = func(call goja.FunctionCall, runtime *goja.Runtime) goja.Value {
		name := call.Argument(0).String()
		var args []any
		for _, arg := range call.Arguments[1:] {
			args = append(args, arg.Export())
		}
		partial, err := p.site.Partial(name, p.jsxGlobals(), args...)
		if err != nil {
			return runtime.ToValue(err.Error())
		}
		return runtime.ToValue(string(partial))
	}
	g["pages"] = func(call goja.FunctionCall, runtime *goja.Runtime) goja.Value {
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
	}
	return g
}

func (p *Page) loadGlobalsDefaults() error {
	globals := make(map[string]any)

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
		g, err := ExportJSX(globalsSrc, globals)
		if err != nil {
			return err
		}
		for k, v := range g {
			globals[k] = v
		}
	}
	p.defaultGlobals = globals
	return nil
}

func (p *Page) loadDataDefaults() error {
	// TODO: load site defaults

	// find all _data.jsx files in parent directories
	var dataFiles []string
	dir := filepath.Dir(p.path)
	stop := false
	for {
		dataPath := path.Join(dir, "_data"+ExtJSX)
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
		data, err := ExportJSX(dataSrc, p.jsxGlobals())
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
	for ext := range formats {
		suffixes = append(suffixes, ext)
		suffixes = append(suffixes, "/index"+ext)
	}

	for _, suffix := range suffixes {
		src, err = os.ReadFile(path.Join(p.site.dir, p.path+suffix))
		if err == nil {
			format = filepath.Ext(suffix)
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

	if err := p.loadGlobalsDefaults(); err != nil {
		return err
	}
	if err := p.loadDataDefaults(); err != nil {
		return err
	}

	p.data[Source] = string(src)

	f, ok := formats[format]
	if !ok {
		return fmt.Errorf("unknown page format: %s", format)
	}

	body, data, err := f.Parse(p)
	if err != nil {
		return err
	}
	for k, v := range data {
		p.data[k] = v
	}
	p.data[Body] = string(body)

	return nil
}

func (p *Page) Render(w io.Writer) (err error) {
	if err = p.Parse(); err != nil {
		return
	}

	// Find all layout files in parent directories
	defaultLayouts := []string{}
	dir := filepath.Dir(p.path)
	for dir != "." && dir != "/" {
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

		dir = filepath.Dir(dir)
	}

	if len(defaultLayouts) > 0 && p.data[Layout] == nil {
		p.data[Layout] = defaultLayouts[0]
		defaultLayouts = defaultLayouts[1:]
	}

	out := p.Body()
	for {
		layout, ok := p.data[Layout].(string)
		if !ok {
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
			out, err = RenderJSX(layoutSrc, p.jsxGlobals(), string(out))
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
			funcs := p.templateFuncs()
			funcs["content"] = func() template.HTML {
				return template.HTML(string(out))
			}
			out, err = RenderTemplate(layoutPath, rest, funcs)
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
	_, err = w.Write(formatTags(out))
	return
}

func formatTags(input []byte) []byte {
	formatted := gohtml.Format(string(input))
	return []byte(formatted)
}
