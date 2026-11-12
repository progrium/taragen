package taragen

import (
	"fmt"
	"math"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/dop251/goja"
	"github.com/progrium/taragen/internal/raster"
)

// siteFonts is the font database raster outputs and measureText use, built
// on first use from every font file in the site.
type siteFonts struct {
	fonts *raster.Fonts
	sig   string // the font files' paths, sizes, and times when loaded
}

// fontSignature identifies the site's font files as they are now, so a
// change to any of them can be noticed.
func (s *Site) fontSignature() (string, error) {
	paths, err := raster.FindFonts(s.dir)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, p := range paths {
		fi, err := os.Stat(filepath.Join(s.dir, filepath.FromSlash(p)))
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "%s %d %d\n", p, fi.Size(), fi.ModTime().UnixNano())
	}
	return b.String(), nil
}

// Fonts returns the site's font database, loading it on first use.
func (s *Site) Fonts() (*raster.Fonts, error) {
	s.fontsMu.Lock()
	defer s.fontsMu.Unlock()
	if s.fonts.fonts != nil {
		return s.fonts.fonts, nil
	}
	sig, err := s.fontSignature()
	if err != nil {
		return nil, err
	}
	fonts, err := raster.LoadSiteFonts(s.dir)
	if err != nil {
		return nil, fmt.Errorf("loading site fonts: %w", err)
	}
	s.fonts = siteFonts{fonts: fonts, sig: sig}
	return fonts, nil
}

// refreshFonts drops a loaded font database if any font file changed since
// it was loaded, so the next use rebuilds it. ParseAll calls it.
func (s *Site) refreshFonts() error {
	s.fontsMu.Lock()
	defer s.fontsMu.Unlock()
	if s.fonts.fonts == nil {
		return nil
	}
	sig, err := s.fontSignature()
	if err != nil {
		return err
	}
	if sig != s.fonts.sig {
		s.fonts.fonts.Close()
		s.fonts = siteFonts{}
	}
	return nil
}

// isPNG reports whether a content type is image/png, ignoring parameters.
func isPNG(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	return err == nil && mediaType == "image/png"
}

// rasterize renders a raster output's SVG to PNG with the site's fonts and
// the scale from the output's data.
func (s *Site) rasterize(svg []byte, data Data) ([]byte, error) {
	scale := 1
	if v, ok := data["scale"]; ok {
		f, ok := toFloat(v)
		if !ok || f != math.Trunc(f) || f < 1 || f > raster.MaxScale {
			return nil, fmt.Errorf("scale must be a whole number from 1 to %d, got %v", raster.MaxScale, v)
		}
		scale = int(f)
	}
	fonts, err := s.Fonts()
	if err != nil {
		return nil, err
	}
	return raster.Render(svg, fonts, raster.Options{Scale: scale, SiteDir: s.dir})
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case float64:
		return n, true
	}
	return 0, false
}

// measureTextFunc is the measureText global: the advance width of a string
// in pixels, from the same fonts and shaping as raster outputs.
func measureTextFunc(s *Site) func(goja.FunctionCall, *goja.Runtime) goja.Value {
	return func(call goja.FunctionCall, rt *goja.Runtime) goja.Value {
		throw := func(err error) {
			panic(rt.NewGoError(fmt.Errorf("measureText: %w", err)))
		}
		text := call.Argument(0)
		if goja.IsUndefined(text) || goja.IsNull(text) {
			throw(fmt.Errorf("text is required"))
		}
		var style raster.TextStyle
		if opts := call.Argument(1); !goja.IsUndefined(opts) && !goja.IsNull(opts) {
			o := opts.ToObject(rt)
			get := func(key string) goja.Value {
				v := o.Get(key)
				if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
					return nil
				}
				return v
			}
			if v := get("family"); v != nil {
				style.Family = v.String()
			}
			if v := get("size"); v != nil {
				style.Size = v.ToFloat()
			}
			if v := get("weight"); v != nil {
				style.Weight = int(v.ToInteger())
			}
			if v := get("style"); v != nil {
				style.Style = v.String()
			}
			if v := get("letterSpacing"); v != nil {
				style.LetterSpacing = v.ToFloat()
			}
		}
		fonts, err := s.Fonts()
		if err != nil {
			throw(err)
		}
		w, err := fonts.Measure(text.String(), style)
		if err != nil {
			// Measure already says measureText.
			panic(rt.NewGoError(err))
		}
		return rt.ToValue(w)
	}
}
