package taragen

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
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
)

type Page struct {
	path    string // normalized path to the page (ex: path/to/page)
	data    Data
	site    *Site
	globals map[string]any
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

func getData[T any](data Data, key string, fallback T) T {
	if data[key] == nil {
		return fallback
	}
	return data[key].(T)
}

func (p *Page) Body() []byte {
	return []byte(getData[string](p.data, Body, ""))
}

func (p *Page) Source() []byte {
	return []byte(getData[string](p.data, Source, ""))
}

func (p *Page) IsDir() bool {
	return getData[bool](p.data, IsDir, false)
}

func (p *Page) Date() string {
	return getData[string](p.data, Date, "")
}

func (p *Page) Slug() string {
	return getData[string](p.data, Slug, "")
}

func (p *Page) ContentType() string {
	return getData[string](p.data, ContentType, "text/html")
}

func (p *Page) Subpages() (subpages []*Page) {
	for _, page := range p.site.pages {
		if strings.HasPrefix(page.path, p.path+"/") || p.path == "." {
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

func (p *Page) loadGlobals() error {
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
		data, err := ExportJSX(dataSrc, p.globals)
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

	for _, suffix := range suffixes {
		src, err = os.ReadFile(path.Join(p.site.dir, p.path+suffix))
		if err == nil {
			format = filepath.Ext(suffix)
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

	origLayout := p.data[Layout]
	defer func() {
		p.data[Layout] = origLayout
	}()

	// Find all layout files in parent directories
	defaultLayouts := []string{}
	if origLayout == nil {
		dir := filepath.Dir(p.path)
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
			out, err = RenderJSX(layoutSrc, p.globals, string(out))
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
	if p.ContentType() == "text/html" {
		out = gohtml.FormatBytes(out)
	}
	_, err = w.Write(out)
	return
}
