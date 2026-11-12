package taragen

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/progrium/taragen/internal/raster"
)

func testFont(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("internal/raster/testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func decodePNG(t *testing.T, b string) image.Image {
	t.Helper()
	img, err := png.Decode(strings.NewReader(b))
	if err != nil {
		t.Fatalf("not a PNG: %v", err)
	}
	return img
}

func px(img image.Image, x, y int) color.NRGBA {
	return color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
}

var (
	white = color.NRGBA{255, 255, 255, 255}
	black = color.NRGBA{0, 0, 0, 255}
)

// cardTemplate draws a white card, a black bar, and the page title in
// Silkscreen.
const cardTemplate = `const W = 1200, H = 630;
<svg width={W} height={H} viewBox={` + "`0 0 ${W} ${H}`" + `}>
  <rect width={W} height={H} fill="#fff" />
  <rect x="80" y="520" width="1040" height="8" fill="#000" />
  <text x="80" y="200" font-family="Silkscreen" font-size="64">{page.title ?? "home"}</text>
</svg>`

func TestCardPNG(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"_assets/fonts/Silkscreen-Regular.ttf": testFont(t, "ttf/Silkscreen-Regular.ttf"),
		"_outputs/card.png.jsx":                cardTemplate,
		"index.jsx":                            `<p/>`,
		"about.md":                             "---\ntitle: About\n---\nabout\n",
	})
	mustBuild(t, dir)
	for _, name := range []string{"about.card.png", "index.card.png"} {
		img := decodePNG(t, readOut(t, dir, name))
		if b := img.Bounds(); b.Dx() != 1200 || b.Dy() != 630 {
			t.Errorf("%s: size %v", name, b)
		}
		if got := px(img, 10, 10); got != white {
			t.Errorf("%s: background %v", name, got)
		}
		if got := px(img, 600, 524); got != black {
			t.Errorf("%s: bar %v", name, got)
		}
		ink := 0
		for x := 80; x < 1100; x++ {
			for y := 130; y < 210; y++ {
				if px(img, x, y) != white {
					ink++
				}
			}
		}
		if ink == 0 {
			t.Errorf("%s: no title drawn", name)
		}
	}
	if readOut(t, dir, "about.card.png") == readOut(t, dir, "index.card.png") {
		t.Error("cards for different titles are identical")
	}
}

func TestCardSVGString(t *testing.T) {
	svg := `<svg width="40" height="20"><rect width="40" height="20" fill="#fff"/><rect width="20" height="20" fill="#00f"/></svg>`
	dir := writeSite(t, map[string]string{
		"_outputs/el.png.jsx":  svg,
		"_outputs/str.png.jsx": "`" + svg + "`",
		"index.jsx":            `<p/>`,
	})
	mustBuild(t, dir)
	el, str := readOut(t, dir, "index.el.png"), readOut(t, dir, "index.str.png")
	if el != str {
		t.Error("string and element render differently")
	}
	if got := px(decodePNG(t, el), 5, 5); got != (color.NRGBA{0, 0, 255, 255}) {
		t.Errorf("pixel %v", got)
	}
}

func TestCardScale(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"_outputs/card.png.jsx": `data({scale: 2}); <svg width="1200" height="630"><rect width="600" height="630" fill="#000"/></svg>`,
		"index.jsx":             `<p/>`,
	})
	mustBuild(t, dir)
	img := decodePNG(t, readOut(t, dir, "index.card.png"))
	if b := img.Bounds(); b.Dx() != 2400 || b.Dy() != 1260 {
		t.Errorf("size %v", b)
	}
	if got := px(img, 1190, 1250); got != black {
		t.Errorf("scaled pixel %v", got)
	}

	for name, tmpl := range map[string]string{
		"scale 5":    `data({scale: 5}); <svg width="10" height="10"/>`,
		"scale 1.5":  `data({scale: 1.5}); <svg width="10" height="10"/>`,
		"scale text": `data({scale: "2"}); <svg width="10" height="10"/>`,
		"width 5000": `<svg width="5000" height="10"/>`,
		"no height":  `<svg width="10"/>`,
		"not svg":    `<p>hi</p>`,
		"plain text": "`hello`",
		"bad markup": "`<svg width=\"1\" height=\"1\"><g></svg>`",
	} {
		dir := writeSite(t, map[string]string{"_outputs/card.png.jsx": tmpl, "index.jsx": `<p/>`})
		err := buildQuiet(t, dir)
		if err == nil || !strings.Contains(err.Error(), "output card of /") {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}

func TestCardFonts(t *testing.T) {
	svg := `<svg width="300" height="40"><rect width="300" height="40" fill="#fff"/><text x="8" y="32" font-family="Silkscreen" font-size="32">HELLO</text></svg>`
	var cards []string
	for _, font := range []string{"ttf/Silkscreen-Regular.ttf", "woff2/Silkscreen-Regular.woff2", "woff/Silkscreen-Regular.woff"} {
		dir := writeSite(t, map[string]string{
			"fonts/" + filepath.Base(font): testFont(t, font),
			"_outputs/card.png.jsx":        svg,
			"index.jsx":                    `<p/>`,
		})
		mustBuild(t, dir)
		cards = append(cards, readOut(t, dir, "index.card.png"))
	}
	if cards[0] != cards[1] || cards[0] != cards[2] {
		t.Error("TTF, WOFF2, and WOFF render differently")
	}

	// Fonts under _public/ and dot directories aren't loaded, and system
	// fonts never are, so with only those the family is unknown.
	dir := writeSite(t, map[string]string{
		"_public/Silkscreen-Regular.ttf": testFont(t, "ttf/Silkscreen-Regular.ttf"),
		".hidden/Silkscreen-Regular.ttf": testFont(t, "ttf/Silkscreen-Regular.ttf"),
		"_outputs/card.png.jsx":          strings.Replace(svg, "Silkscreen", "Silkscreen, DejaVu Sans, Arial, Helvetica, sans-serif, serif", 1),
		"index.jsx":                      `<p/>`,
	})
	if err := buildQuiet(t, dir); err == nil || !strings.Contains(err.Error(), "loaded families: none") {
		t.Errorf("err = %v", err)
	}
}

func TestCardUnknownFont(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"_assets/Silkscreen-Regular.ttf": testFont(t, "ttf/Silkscreen-Regular.ttf"),
		"_outputs/card.png.jsx":          `<svg width="10" height="10"><text font-family="Silkscrene">x</text></svg>`,
		"index.jsx":                      `<p/>`,
	})
	err := buildQuiet(t, dir)
	if err == nil || !strings.Contains(err.Error(), "Silkscrene") || !strings.Contains(err.Error(), "loaded families: Silkscreen") {
		t.Errorf("err = %v", err)
	}
}

func TestCardImages(t *testing.T) {
	red := testFont(t, "img/red.png")
	tmpl := func(href string) string {
		return `<svg width="4" height="4"><image width="4" height="4" href="` + href + `" /></svg>`
	}
	ok := map[string]string{
		"site": "/_assets/red.png",
		"data": "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte(red)),
	}
	for name, href := range ok {
		dir := writeSite(t, map[string]string{"_assets/red.png": red, "_outputs/card.png.jsx": tmpl(href), "index.jsx": `<p/>`})
		mustBuild(t, dir)
		if got := px(decodePNG(t, readOut(t, dir, "index.card.png")), 2, 2); got != (color.NRGBA{255, 0, 0, 255}) {
			t.Errorf("%s: pixel %v", name, got)
		}
	}

	outside := t.TempDir()
	os.WriteFile(filepath.Join(outside, "secret.png"), []byte(red), 0o644)
	for name, href := range map[string]string{
		"https":    "https://example.com/red.png",
		"file":     "file://" + filepath.Join(outside, "secret.png"),
		"relative": "_assets/red.png",
		"dotdot":   "/_assets/../../" + filepath.Base(outside) + "/secret.png",
		"symlink":  "/_assets/link.png",
	} {
		dir := writeSite(t, map[string]string{"_assets/red.png": red, "_outputs/card.png.jsx": tmpl(href), "index.jsx": `<p/>`})
		os.Symlink(filepath.Join(outside, "secret.png"), filepath.Join(dir, "_assets", "link.png"))
		err := buildQuiet(t, dir)
		if err == nil || !strings.Contains(err.Error(), "image") {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}

func TestCardDeterministic(t *testing.T) {
	files := map[string]string{
		"_assets/Silkscreen-Regular.ttf": testFont(t, "ttf/Silkscreen-Regular.ttf"),
		"_outputs/card.png.jsx":          cardTemplate,
		"index.jsx":                      `<p/>`,
	}
	a, b := writeSite(t, files), writeSite(t, files)
	mustBuild(t, a)
	mustBuild(t, b)
	if readOut(t, a, "index.card.png") != readOut(t, b, "index.card.png") {
		t.Error("two builds differ")
	}
}

func TestMeasureText(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"_assets/Silkscreen-Regular.ttf": testFont(t, "ttf/Silkscreen-Regular.ttf"),
		"_assets/Silkscreen-Bold.ttf":    testFont(t, "ttf/Silkscreen-Bold.ttf"),
		"_globals.jsx": `wrap = (text, font, width) => text.split(" ").reduce((lines, word) => {
  const last = lines[lines.length - 1];
  if (last && measureText(` + "`${last} ${word}`" + `, font) <= width) lines[lines.length - 1] = ` + "`${last} ${word}`" + `;
  else lines.push(word);
  return lines;
}, []);`,
		"index.jsx": `<p>{[
  measureText("HELLO", {family: "Silkscreen", size: 16}),
  measureText("HELLO", {family: "Silkscreen", size: 16, weight: 700}),
  measureText("HELLO", {family: "Silkscreen"}),
  measureText("HELLO", {family: "Silkscreen", size: 16, letterSpacing: 2}),
  wrap("one two three four five six", {family: "Silkscreen", size: 16}, 120).join("|"),
].join(",")}</p>`,
	})
	site := parsed(t, dir, false)
	var buf bytes.Buffer
	if err := site.Page(".").Render(&buf); err != nil {
		t.Fatal(err)
	}
	fonts, err := site.Fonts()
	if err != nil {
		t.Fatal(err)
	}
	w, _ := fonts.Measure("HELLO", raster.TextStyle{Family: "Silkscreen", Size: 16})
	if w == 0 {
		t.Fatal("zero width")
	}
	_, body, _ := strings.Cut(buf.String(), "<p>")
	body, _, _ = strings.Cut(body, "</p>")
	parts := strings.Split(body, ",")
	if len(parts) != 5 {
		t.Fatalf("got %q", buf.String())
	}
	if parts[0] == parts[1] {
		t.Errorf("weight didn't change the width: %v", parts)
	}
	if parts[0] != parts[2] {
		t.Errorf("default size isn't 16: %v", parts)
	}
	if parts[0] == parts[3] {
		t.Errorf("letterSpacing ignored: %v", parts)
	}
	if parts[4] != "one two|three four|five six" {
		t.Errorf("wrap = %q", parts[4])
	}

	for name, page := range map[string]string{
		"unknown family": `<p>{measureText("x", {family: "Nope"})}</p>`,
		"no family":      `<p>{measureText("x")}</p>`,
	} {
		dir := writeSite(t, map[string]string{"_assets/a.ttf": testFont(t, "ttf/Silkscreen-Regular.ttf"), "index.jsx": page})
		if err := buildQuiet(t, dir); err == nil || !strings.Contains(err.Error(), "measureText") {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}

func TestServeCard(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"_assets/fonts/card.ttf": testFont(t, "ttf/Silkscreen-Regular.ttf"),
		"_outputs/card.png.jsx":  cardTemplate,
		"about.md":               "---\ntitle: About\n---\nabout\n",
	})
	site := parsed(t, dir, false)
	get := func() string {
		t.Helper()
		rec := serve(t, site, "/about.card.png")
		if rec.Code != 200 || rec.Header().Get("Content-Type") != "image/png" {
			t.Fatalf("status %d, type %q: %s", rec.Code, rec.Header().Get("Content-Type"), rec.Body.String())
		}
		decodePNG(t, rec.Body.String())
		return rec.Body.String()
	}
	first := get()

	// A different font under the same path changes the card after the
	// reparse the watcher does.
	os.WriteFile(filepath.Join(dir, "_assets/fonts/card.ttf"), []byte(testFont(t, "ttf/Silkscreen-Bold.ttf")), 0o644)
	if err := site.ParseAll(); err != nil {
		t.Fatal(err)
	}
	second := get()
	if second == first {
		t.Error("changing the font file didn't change the card")
	}

	// So does editing the template, with no reparse.
	os.WriteFile(filepath.Join(dir, "_outputs/card.png.jsx"), []byte(strings.Replace(cardTemplate, `fill="#000"`, `fill="#f00"`, 1)), 0o644)
	if third := get(); third == second {
		t.Error("editing the template didn't change the card")
	}

	os.WriteFile(filepath.Join(dir, "_outputs/card.png.jsx"), []byte(`<svg width="99999" height="1"/>`), 0o644)
	if rec := serve(t, site, "/about.card.png"); rec.Code != 500 || !strings.Contains(rec.Body.String(), "maximum") {
		t.Errorf("bad card: status %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCardNoEngine(t *testing.T) {
	// Whether the module was compiled is per process, so build in a fresh one.
	if os.Getenv("TARAGEN_TEST_NO_ENGINE") != "" {
		dir := writeSite(t, map[string]string{
			"_assets/Silkscreen-Regular.ttf": testFont(t, "ttf/Silkscreen-Regular.ttf"),
			"_outputs/txt.jsx":               "`txt`",
			"index.jsx":                      `<p/>`,
		})
		mustBuild(t, dir)
		if raster.Compiled() {
			t.Error("a site with no raster outputs compiled the module")
		}
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestCardNoEngine$", "-test.count=1")
	cmd.Env = append(os.Environ(), "TARAGEN_TEST_NO_ENGINE=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
}
