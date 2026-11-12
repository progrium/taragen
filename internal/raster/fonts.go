package raster

import (
	"context"
	"encoding/binary"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
)

// FontExts are the file extensions loaded as fonts.
var FontExts = []string{".ttf", ".otf", ".ttc", ".woff", ".woff2"}

// IsFontFile reports whether path has a font file extension.
func IsFontFile(path string) bool {
	return slices.Contains(FontExts, strings.ToLower(filepath.Ext(path)))
}

// FindFonts returns the font files under dir, as slash-separated paths
// relative to it, sorted. It skips _public/ and any path with a component
// starting with a dot.
func FindFonts(dir string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)
		name := d.Name()
		if d.IsDir() {
			if strings.HasPrefix(name, ".") || rel == "_public" {
				return fs.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(name, ".") || !IsFontFile(name) {
			return nil
		}
		paths = append(paths, rel)
		return nil
	})
	slices.Sort(paths)
	return paths, err
}

// Fonts is a font database: the font files raster outputs and measureText
// can use. It's safe for concurrent use.
type Fonts struct {
	data     [][]byte // plain sfnt data, in path order
	families []string

	mu      sync.Mutex
	measure *instance // long-lived, for Measure
}

// LoadSiteFonts loads every font file FindFonts finds in dir.
func LoadSiteFonts(dir string) (*Fonts, error) {
	paths, err := FindFonts(dir)
	if err != nil {
		return nil, err
	}
	files := make(map[string][]byte, len(paths))
	for _, p := range paths {
		b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(p)))
		if err != nil {
			return nil, err
		}
		files[p] = b
	}
	return LoadFonts(files)
}

// LoadFonts builds a font database from font files keyed by path. Where two
// files provide the same family, weight, and style, the one whose path sorts
// first wins. WOFF and WOFF2 files are decoded once, here.
func LoadFonts(files map[string][]byte) (*Fonts, error) {
	ctx := context.Background()
	in, err := newInstance(ctx)
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}
	slices.Sort(paths)

	f := &Fonts{}
	req := request{}.u32(uint32(len(paths)))
	for _, p := range paths {
		data := files[p]
		if isWOFF(data) {
			data, err = in.call(ctx, "decode", data)
			if err != nil {
				in.close(ctx)
				return nil, fmt.Errorf("%s: %w", p, err)
			}
		}
		f.data = append(f.data, data)
		req = req.blob(data)
	}
	names, err := in.call(ctx, "load_fonts", req)
	if err != nil {
		in.close(ctx)
		return nil, err
	}
	if len(names) > 0 {
		f.families = strings.Split(string(names), "\n")
	}
	f.measure = in
	return f, nil
}

func isWOFF(b []byte) bool {
	return len(b) >= 4 && (string(b[:4]) == "wOFF" || string(b[:4]) == "wOF2")
}

// Families returns the loaded family names.
func (f *Fonts) Families() []string {
	return slices.Clone(f.families)
}

// Close frees the module instance kept for measuring.
func (f *Fonts) Close() {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.measure != nil {
		f.measure.close(context.Background())
		f.measure = nil
	}
}

// fontsRequest is the load_fonts request for this database.
func (f *Fonts) fontsRequest() request {
	req := request{}.u32(uint32(len(f.data)))
	for _, d := range f.data {
		req = req.blob(d)
	}
	return req
}

// TextStyle is the font measureText uses, with CSS meanings and defaults.
type TextStyle struct {
	Family        string
	Size          float64 // pixels, default 16
	Weight        int     // default 400
	Style         string  // normal, italic, or oblique; default normal
	LetterSpacing float64 // pixels, added between letters but not after the last
}

// Measure returns the advance width of text in pixels, laid out and shaped
// the same way the rasterizer would.
func (f *Fonts) Measure(text string, style TextStyle) (float64, error) {
	if style.Family == "" {
		return 0, fmt.Errorf("measureText: a font family is required")
	}
	if style.Size == 0 {
		style.Size = 16
	}
	if style.Weight == 0 {
		style.Weight = 400
	}
	if style.Style == "" {
		style.Style = "normal"
	}
	if style.Size < 0 || style.Weight < 1 || style.Weight > 1000 {
		return 0, fmt.Errorf("measureText: invalid size or weight")
	}

	req := request{}.str(text).str(style.Family)
	req = req.u32(math.Float32bits(float32(style.Size)))
	req = req.u32(uint32(style.Weight))
	req = req.str(style.Style)
	req = req.u32(math.Float32bits(float32(style.LetterSpacing)))

	f.mu.Lock()
	defer f.mu.Unlock()
	ctx := context.Background()
	if f.measure == nil {
		in, err := newInstance(ctx)
		if err != nil {
			return 0, err
		}
		if _, err := in.call(ctx, "load_fonts", f.fontsRequest()); err != nil {
			in.close(ctx)
			return 0, err
		}
		f.measure = in
	}
	out, err := f.measure.call(ctx, "measure", req)
	if err != nil {
		if isTrap(err) {
			f.measure.close(ctx)
			f.measure = nil
		}
		return 0, fmt.Errorf("measureText: %w", err)
	}
	// The module works in f32; round off its noise so whole-pixel widths
	// come out whole.
	w := float64(math.Float32frombits(binary.LittleEndian.Uint32(out)))
	return math.Round(w*1e4) / 1e4, nil
}
