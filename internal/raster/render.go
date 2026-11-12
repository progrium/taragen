package raster

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	// MaxSize is the largest width or height a card may declare.
	MaxSize = 4096
	// MaxScale is the largest scale a card may render at.
	MaxScale = 4
)

const svgNS = "http://www.w3.org/2000/svg"

// Options are the settings for one render.
type Options struct {
	// Scale renders at a whole-number multiple of the SVG's size. Zero means 1.
	Scale int
	// SiteDir is the directory root-relative image paths resolve against.
	// With none, only data: URIs are allowed.
	SiteDir string
}

// Render rasterizes svg to PNG bytes with the given fonts. Each call runs in
// a fresh module instance.
func Render(svg []byte, fonts *Fonts, opts Options) ([]byte, error) {
	scale := opts.Scale
	if scale == 0 {
		scale = 1
	}
	if scale < 1 || scale > MaxScale {
		return nil, fmt.Errorf("scale must be a whole number from 1 to %d, got %d", MaxScale, scale)
	}
	svg, hrefs, err := inspect(svg)
	if err != nil {
		return nil, err
	}

	req := request{}.u32(uint32(scale)).blob(svg)
	req = req.u32(uint32(len(hrefs)))
	for _, href := range hrefs {
		data, err := resolveImage(opts.SiteDir, href)
		if err != nil {
			return nil, err
		}
		req = req.str(href).blob(data)
	}

	ctx := context.Background()
	in, err := newInstance(ctx)
	if err != nil {
		return nil, err
	}
	defer in.close(ctx)
	if fonts != nil {
		if _, err := in.call(ctx, "load_fonts", fonts.fontsRequest()); err != nil {
			return nil, err
		}
	}
	px, err := in.call(ctx, "render", req)
	if isTrap(err) {
		return nil, fmt.Errorf("rendering SVG failed, possibly out of memory: %w", err)
	}
	if err != nil {
		return nil, err
	}
	if len(px) < 8 {
		return nil, errors.New("rendering SVG: short response")
	}
	w := int(binary.LittleEndian.Uint32(px))
	h := int(binary.LittleEndian.Uint32(px[4:]))
	return encodePNG(w, h, px[8:])
}

func isTrap(err error) bool {
	return errors.Is(err, errTrap)
}

// inspect checks that svg's root is an <svg> with a valid pixel size, adds
// the SVG namespace to it if it's missing, and returns the non-data hrefs of
// its <image> elements.
func inspect(svg []byte) ([]byte, []string, error) {
	dec := xml.NewDecoder(bytes.NewReader(svg))
	dec.Strict = true
	var hrefs []string
	seen := map[string]bool{}
	root := true
	for {
		tok, err := dec.RawToken()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("parsing SVG: %w", err)
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if root {
			root = false
			if se.Name.Local != "svg" {
				return nil, nil, fmt.Errorf("a PNG output must return an <svg> element, got <%s>", qname(se.Name))
			}
			if err := checkSize(se); err != nil {
				return nil, nil, err
			}
			if !hasNS(se) {
				svg = addNS(svg, dec, se)
			}
			continue
		}
		if se.Name.Local != "image" {
			continue
		}
		for _, a := range se.Attr {
			if a.Name.Local != "href" {
				continue
			}
			href := strings.TrimSpace(a.Value)
			if strings.HasPrefix(href, "data:") || seen[href] {
				continue
			}
			seen[href] = true
			hrefs = append(hrefs, href)
		}
	}
	if root {
		return nil, nil, errors.New("a PNG output must return an <svg> element, got nothing")
	}
	return svg, hrefs, nil
}

func qname(n xml.Name) string {
	if n.Space != "" {
		return n.Space + ":" + n.Local
	}
	return n.Local
}

func checkSize(se xml.StartElement) error {
	for _, dim := range []string{"width", "height"} {
		v, ok := attr(se, dim)
		if !ok {
			return fmt.Errorf("the root <svg> needs a %s in pixels", dim)
		}
		n, err := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSpace(v), "px"), 64)
		if err != nil || n <= 0 || n != float64(int(n)) {
			return fmt.Errorf("the root <svg> %s must be a whole number of pixels, got %q", dim, v)
		}
		if n > MaxSize {
			return fmt.Errorf("the root <svg> %s is %v, more than the maximum of %d", dim, n, MaxSize)
		}
	}
	return nil
}

func attr(se xml.StartElement, name string) (string, bool) {
	for _, a := range se.Attr {
		if a.Name.Space == "" && a.Name.Local == name {
			return a.Value, true
		}
	}
	return "", false
}

func hasNS(se xml.StartElement) bool {
	for _, a := range se.Attr {
		if a.Name.Space == "" && a.Name.Local == "xmlns" {
			return true
		}
	}
	return false
}

// addNS inserts the SVG namespace right after "<svg" in the root tag.
func addNS(svg []byte, dec *xml.Decoder, se xml.StartElement) []byte {
	i := bytes.Index(svg, []byte("<"+qname(se.Name)))
	if i < 0 || int64(i) > dec.InputOffset() {
		return svg
	}
	i += len(qname(se.Name)) + 1
	out := make([]byte, 0, len(svg)+len(svgNS)+10)
	out = append(out, svg[:i]...)
	out = append(out, ` xmlns="`+svgNS+`"`...)
	return append(out, svg[i:]...)
}

// resolveImage reads an <image> href: a root-relative path to a file inside
// siteDir, after symlinks are resolved. Everything else is an error.
func resolveImage(siteDir, href string) ([]byte, error) {
	fail := func(why string) ([]byte, error) {
		return nil, fmt.Errorf("image %q: %s", href, why)
	}
	if strings.Contains(href, ":") {
		return fail("only data: URIs and root-relative site paths are allowed")
	}
	if !strings.HasPrefix(href, "/") || strings.HasPrefix(href, "//") {
		return fail("paths must be root-relative, like /assets/avatar.png")
	}
	if siteDir == "" {
		return fail("no site directory to resolve it in")
	}
	clean := path.Clean(href)
	if clean != href || strings.Contains(href, "\\") {
		return fail("path must be clean, without . or .. segments")
	}
	root, err := filepath.EvalSymlinks(siteDir)
	if err != nil {
		return nil, err
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	p, err := filepath.EvalSymlinks(filepath.Join(root, filepath.FromSlash(clean)))
	if err != nil {
		return fail("not found in the site")
	}
	rel, err := filepath.Rel(root, p)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return fail("resolves outside the site")
	}
	return os.ReadFile(p)
}
