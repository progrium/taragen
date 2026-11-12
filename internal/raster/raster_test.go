package raster

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func loadFonts(t *testing.T, dir string) *Fonts {
	t.Helper()
	f, err := LoadSiteFonts(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(f.Close)
	return f
}

func render(t *testing.T, svg string, fonts *Fonts, opts Options) image.Image {
	t.Helper()
	b, err := Render([]byte(svg), fonts, opts)
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	return img
}

func rgba(c color.Color) [4]uint8 {
	n := color.NRGBAModel.Convert(c).(color.NRGBA)
	return [4]uint8{n.R, n.G, n.B, n.A}
}

func TestRenderShapes(t *testing.T) {
	svg := `<svg width="40" height="20">
		<rect width="40" height="20" fill="#fff"/>
		<rect x="20" width="20" height="20" fill="#00f"/>
		<rect x="0" y="10" width="10" height="10" fill="#f00" fill-opacity="0.5"/>
	</svg>`
	img := render(t, svg, nil, Options{})
	if b := img.Bounds(); b.Dx() != 40 || b.Dy() != 20 {
		t.Fatalf("size = %v", b)
	}
	if _, ok := img.(*image.NRGBA); !ok {
		t.Errorf("PNG decoded as %T, want RGBA (*image.NRGBA)", img)
	}
	if got := rgba(img.At(5, 5)); got != [4]uint8{255, 255, 255, 255} {
		t.Errorf("white at (5,5) = %v", got)
	}
	if got := rgba(img.At(30, 5)); got != [4]uint8{0, 0, 255, 255} {
		t.Errorf("blue at (30,5) = %v", got)
	}
	if got := rgba(img.At(5, 15)); got[0] != 255 || got[1] < 126 || got[1] > 129 {
		t.Errorf("half red over white at (5,15) = %v", got)
	}
}

func TestRenderTransparent(t *testing.T) {
	img := render(t, `<svg width="4" height="4"><rect width="2" height="4" fill="#0f0" fill-opacity="0.5"/></svg>`, nil, Options{})
	if got := rgba(img.At(0, 0)); got[1] != 255 || got[3] < 127 || got[3] > 128 {
		t.Errorf("translucent green = %v, want unpremultiplied", got)
	}
	if got := rgba(img.At(3, 0)); got[3] != 0 {
		t.Errorf("empty pixel = %v", got)
	}
}

func TestRenderScale(t *testing.T) {
	svg := `<svg width="10" height="5"><rect width="5" height="5" fill="#000"/></svg>`
	img := render(t, svg, nil, Options{Scale: 2})
	if b := img.Bounds(); b.Dx() != 20 || b.Dy() != 10 {
		t.Fatalf("size = %v", b)
	}
	if got := rgba(img.At(9, 9)); got != [4]uint8{0, 0, 0, 255} {
		t.Errorf("scaled rect = %v", got)
	}
	for _, s := range []int{-1, 5} {
		if _, err := Render([]byte(svg), nil, Options{Scale: s}); err == nil {
			t.Errorf("scale %d: no error", s)
		}
	}
}

func TestRenderSizeErrors(t *testing.T) {
	cases := map[string]string{
		`<svg width="5000" height="10"/>`:          "maximum",
		`<svg height="10"/>`:                       "width",
		`<svg width="10" height="50%"/>`:           "whole number",
		`<svg width="10.5" height="10"/>`:          "whole number",
		`<html width="10" height="10"/>`:           "<svg>",
		`hello`:                                    "<svg>",
		`<svg width="10" height="10"><rect></svg>`: "parsing SVG",
	}
	for svg, want := range cases {
		_, err := Render([]byte(svg), nil, Options{})
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: err = %v, want %q", svg, err, want)
		}
	}
	// px units are fine.
	if _, err := Render([]byte(`<svg width="10px" height="10px"/>`), nil, Options{}); err != nil {
		t.Error(err)
	}
}

func TestRenderNamespace(t *testing.T) {
	with := `<svg xmlns="http://www.w3.org/2000/svg" width="4" height="4"><rect width="4" height="4" fill="#f00"/></svg>`
	without := `<svg width="4" height="4"><rect width="4" height="4" fill="#f00"/></svg>`
	a, err := Render([]byte(with), nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	b, err := Render([]byte(without), nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Error("an SVG without xmlns rendered differently")
	}
}

func TestRenderDeterministic(t *testing.T) {
	fonts := loadFonts(t, "testdata/ttf")
	svg := `<svg width="200" height="40"><text x="8" y="24" font-family="Silkscreen" font-size="16">HELLO</text></svg>`
	a, err := Render([]byte(svg), fonts, Options{})
	if err != nil {
		t.Fatal(err)
	}
	b, err := Render([]byte(svg), fonts, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Error("two renders differ")
	}
	for _, chunk := range []string{"tIME", "tEXt", "zTXt", "iTXt"} {
		if bytes.Contains(a, []byte(chunk)) {
			t.Errorf("PNG has a %s chunk", chunk)
		}
	}
}

func inked(img image.Image) int {
	n := 0
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if rgba(img.At(x, y))[3] > 0 {
				n++
			}
		}
	}
	return n
}

func TestFontFormats(t *testing.T) {
	svg := `<svg width="200" height="40"><text x="8" y="24" font-family="Silkscreen" font-size="16">HELLO 123</text></svg>`
	ttf, err := Render([]byte(svg), loadFonts(t, "testdata/ttf"), Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{"testdata/woff", "testdata/woff2"} {
		got, err := Render([]byte(svg), loadFonts(t, dir), Options{})
		if err != nil {
			t.Fatal(dir, err)
		}
		if !bytes.Equal(ttf, got) {
			t.Errorf("%s renders differently from TTF", dir)
		}
	}
	img, _ := png.Decode(bytes.NewReader(ttf))
	if inked(img) == 0 {
		t.Error("no text drawn")
	}
}

func TestUnknownFont(t *testing.T) {
	fonts := loadFonts(t, "testdata/ttf")
	cases := []string{
		`<text font-family="Silkscren">X</text>`,
		`<text font-family="sans-serif">X</text>`,
		`<text>X</text>`,
	}
	for _, c := range cases {
		_, err := Render([]byte(`<svg width="10" height="10">`+c+`</svg>`), fonts, Options{})
		if err == nil || !strings.Contains(err.Error(), "loaded families: Silkscreen") {
			t.Errorf("%s: err = %v", c, err)
		}
	}
	_, err := Render([]byte(`<svg width="10" height="10"><text font-family="Nope">X</text></svg>`), fonts, Options{})
	if err == nil || !strings.Contains(err.Error(), "Nope") {
		t.Errorf("error doesn't name the family: %v", err)
	}
	// A list with one loaded family is fine.
	if _, err := Render([]byte(`<svg width="10" height="10"><text font-family="Nope, Silkscreen">X</text></svg>`), fonts, Options{}); err != nil {
		t.Error(err)
	}
	// So is an SVG with no text and no fonts at all.
	if _, err := Render([]byte(`<svg width="10" height="10"/>`), nil, Options{}); err != nil {
		t.Error(err)
	}
}

func TestFindFonts(t *testing.T) {
	dir := t.TempDir()
	for _, p := range []string{
		"a.ttf", "fonts/b.WOFF2", "fonts/c.otf", "d.woff", "e.ttc", "notes.txt",
		"_public/f.ttf", ".git/g.ttf", "x/.h.ttf", "_assets/i.ttf",
	} {
		full := filepath.Join(dir, p)
		os.MkdirAll(filepath.Dir(full), 0o755)
		os.WriteFile(full, nil, 0o644)
	}
	got, err := FindFonts(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := "_assets/i.ttf a.ttf d.woff e.ttc fonts/b.WOFF2 fonts/c.otf"
	if strings.Join(got, " ") != want {
		t.Errorf("FindFonts = %v, want %s", got, want)
	}
}

func TestFontPrecedence(t *testing.T) {
	ttf, _ := os.ReadFile("testdata/ttf/Silkscreen-Regular.ttf")
	bold, _ := os.ReadFile("testdata/ttf/Silkscreen-Bold.ttf")
	svg := `<svg width="200" height="40"><text x="8" y="24" font-family="Silkscreen" font-size="16">HELLO</text></svg>`
	want, err := Render([]byte(svg), mustLoad(t, map[string][]byte{"a.ttf": ttf}), Options{})
	if err != nil {
		t.Fatal(err)
	}
	// A second regular face under a later path doesn't win.
	f := mustLoad(t, map[string][]byte{"b/second.ttf": ttf, "a/first.ttf": ttf, "z.ttf": bold})
	got, err := Render([]byte(svg), f, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Error("duplicate faces changed the result")
	}
	if fams := f.Families(); len(fams) != 1 || fams[0] != "Silkscreen" {
		t.Errorf("Families = %v", fams)
	}
}

func mustLoad(t *testing.T, files map[string][]byte) *Fonts {
	t.Helper()
	f, err := LoadFonts(files)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(f.Close)
	return f
}

func TestBadFont(t *testing.T) {
	if _, err := LoadFonts(map[string][]byte{"bad.woff2": []byte("wOF2garbage")}); err == nil || !strings.Contains(err.Error(), "bad.woff2") {
		t.Errorf("err = %v", err)
	}
}

func TestImages(t *testing.T) {
	site := t.TempDir()
	red, _ := os.ReadFile("testdata/img/red.png")
	os.MkdirAll(filepath.Join(site, "assets"), 0o755)
	os.WriteFile(filepath.Join(site, "assets", "red.png"), red, 0o644)
	outside := t.TempDir()
	os.WriteFile(filepath.Join(outside, "secret.png"), red, 0o644)
	if err := os.Symlink(filepath.Join(outside, "secret.png"), filepath.Join(site, "assets", "link.png")); err != nil {
		t.Fatal(err)
	}
	os.Symlink(filepath.Join(site, "assets", "red.png"), filepath.Join(site, "assets", "inside.png"))

	card := func(href string) string {
		return `<svg width="4" height="4" xmlns:xlink="http://www.w3.org/1999/xlink"><image width="4" height="4" href="` + href + `"/></svg>`
	}
	dataURI := "data:image/png;base64," + base64.StdEncoding.EncodeToString(red)
	for _, href := range []string{"/assets/red.png", "/assets/inside.png", dataURI} {
		img := render(t, card(href), nil, Options{SiteDir: site})
		if got := rgba(img.At(2, 2)); got != [4]uint8{255, 0, 0, 255} {
			t.Errorf("%.40s: pixel = %v", href, got)
		}
	}
	xlink := `<svg width="4" height="4" xmlns:xlink="http://www.w3.org/1999/xlink"><image width="4" height="4" xlink:href="/assets/red.png"/></svg>`
	if got := rgba(render(t, xlink, nil, Options{SiteDir: site}).At(2, 2)); got != [4]uint8{255, 0, 0, 255} {
		t.Errorf("xlink:href pixel = %v", got)
	}

	for _, href := range []string{
		"https://example.com/a.png",
		"file:///etc/passwd",
		"assets/red.png",
		"/assets/../assets/red.png",
		"/../" + filepath.Base(outside) + "/secret.png",
		"/assets/link.png",
		"/assets/missing.png",
		"//example.com/a.png",
	} {
		_, err := Render([]byte(card(href)), nil, Options{SiteDir: site})
		if err == nil || !strings.Contains(err.Error(), "image") {
			t.Errorf("%s: err = %v", href, err)
		}
	}
}

func TestMeasure(t *testing.T) {
	fonts := loadFonts(t, "testdata/ttf")
	style := TextStyle{Family: "Silkscreen", Size: 16}
	w, err := fonts.Measure("HELLO", style)
	if err != nil {
		t.Fatal(err)
	}
	// Silkscreen draws on an 8 by 8 grid, so advances are whole pixels at 16px.
	if w <= 0 || w != float64(int(w)) {
		t.Errorf("width = %v", w)
	}

	// The measured width matches the rendered extent within a pixel.
	svg := `<svg width="400" height="40"><text x="10" y="24" font-family="Silkscreen" font-size="16" text-rendering="optimizeSpeed">HELLO</text></svg>`
	img := render(t, svg, fonts, Options{})
	right := 0
	for x := 0; x < 400; x++ {
		for y := 0; y < 40; y++ {
			if rgba(img.At(x, y))[3] > 0 {
				right = x + 1
			}
		}
	}
	// The ink stops short of the advance by the glyph's right side bearing.
	if ink := float64(right - 10); ink > w || w-ink > 2*16/8+1 {
		t.Errorf("ink width %v vs measured %v", ink, w)
	}

	if double, _ := fonts.Measure("HELLOHELLO", style); double != 2*w {
		t.Errorf("twice the text = %v, want %v", double, 2*w)
	}
	if big, _ := fonts.Measure("HELLO", TextStyle{Family: "Silkscreen", Size: 32}); big != 2*w {
		t.Errorf("twice the size = %v, want %v", big, 2*w)
	}
	if sp, _ := fonts.Measure("HELLO", TextStyle{Family: "Silkscreen", Size: 16, LetterSpacing: 2}); sp != w+8 {
		// Spacing goes between the five letters, not after the last.
		t.Errorf("letter spacing 2 = %v, want %v", sp, w+8)
	}
	if def, _ := fonts.Measure("HELLO", TextStyle{Family: "Silkscreen"}); def != w {
		t.Errorf("default size = %v, want 16px width %v", def, w)
	}
	bold, err := fonts.Measure("HELLO", TextStyle{Family: "Silkscreen", Size: 16, Weight: 700})
	if err != nil || bold == w {
		t.Errorf("bold = %v (%v), want different from %v", bold, err, w)
	}
	if sp, _ := fonts.Measure(" HELLO ", style); sp <= w {
		t.Errorf("surrounding spaces were dropped: %v", sp)
	}
	if empty, err := fonts.Measure("", style); empty != 0 || err != nil {
		t.Errorf("empty = %v, %v", empty, err)
	}
	if esc, err := fonts.Measure(`<a & "b">`, style); esc <= 0 || err != nil {
		t.Errorf("markup characters = %v, %v", esc, err)
	}
	if _, err := fonts.Measure("x", TextStyle{Family: "Nope"}); err == nil || !strings.Contains(err.Error(), "Nope") {
		t.Errorf("unknown family: %v", err)
	}
	if _, err := fonts.Measure("x", TextStyle{}); err == nil {
		t.Error("no family: no error")
	}
}

func TestLazyCompile(t *testing.T) {
	// Finding fonts doesn't need the module. This only holds when it runs
	// first, so it checks FindFonts alone rather than the process state.
	before := compiles
	if _, err := FindFonts("testdata"); err != nil {
		t.Fatal(err)
	}
	if compiles != before {
		t.Error("FindFonts compiled the module")
	}
	if Compiled() != (compiles > 0) {
		t.Error("Compiled disagrees with the counter")
	}
}
