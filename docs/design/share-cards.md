---
slug: share-cards
title: Share card images
status: draft
created: 2026-11-12
authors: [progrium]
depends: [page-outputs]
areas: [jsx, site, serve, deps]
branch: []
---

# Proposal: Share card images

## Abstract

Let a site make a PNG share card for every page, for `og:image` and similar tags.
A card is an ordinary page output, such as `_outputs/card.png.jsx` giving `/foo.card.png`, whose template returns SVG.
When an output's content type is `image/png`, taragen rasterizes the SVG in-process with the site's own font files, so `build` writes PNGs and `serve` renders them on request.
The rasterizer is resvg compiled to WebAssembly and run by wazero, which keeps taragen a single pure-Go binary on every platform it ships for.
A new `measureText()` global uses the same engine, so templates can wrap titles to fit.

## Background

Social sites and chat apps fetch the image named by `og:image` (and `twitter:image`) to preview a link, and they don't accept SVG: they want PNG or JPEG.
The progrium.xyz redesign wants a generated card for every page, in the site's black and white style, set in Recursive and Silkscreen.
Today the site's layout points every page at one static `site_cover` image.

The [page-outputs](page-outputs.md) proposal gives each page extra output files from `_outputs/` templates, and lists share cards as a follow-on: "an output with `contentType: image/png` whose template returns SVG or HTML for taragen to rasterize."
It also leaves open whether an output may return bytes, deferring that to this proposal.
What's missing is the rasterizer: something inside the taragen binary that turns markup and fonts into a PNG.

Taragen is one Go binary with no Node or browser dependency (see Key design decisions in the [spec](../spec.md)), and `make dist` cross-compiles it for darwin, windows, and linux on amd64 and arm64, plus linux-386 in CI.
Any rasterizer has to fit that: no cgo, no system libraries, no external programs.

## Proposal

### PNG outputs

An output whose content type is `image/png` is a raster output.
With the page-outputs suffix rule, any template named `_outputs/<name>.png.jsx` is one by default, and `data({ contentType: "image/png" })` makes any output one.

A raster output's template returns SVG, either as an element (usually a root `<svg>`) or as a string of SVG markup.
Taragen serializes an element the same way it serializes HTML, adds `xmlns="http://www.w3.org/2000/svg"` to the root if it's missing, rasterizes the result, and writes or serves the PNG bytes.
The template never handles bytes, so outputs still return only strings and elements.

The root `<svg>` must have numeric `width` and `height` attributes in pixels, which set the PNG's size.
Neither may exceed 4096.
`data({ scale: 2 })` renders at a whole-number multiple of that size (default `1`, at most `4`), for sharper cards on dense screens without changing the template's coordinates.

Rendering follows SVG 1.1 static rendering plus the parts of SVG 2 that resvg supports: shapes, paths, gradients, patterns, clipping, masks, filters, `<text>` and `<tspan>`, and `<image>`.
Scripts, animation, and `<foreignObject>` are ignored.

The PNG is a non-interlaced RGBA image with no timestamps or text chunks, so the same SVG and fonts always produce the same bytes.

### Fonts

Raster outputs see only the site's own fonts.
Every font file in the site directory (`.ttf`, `.otf`, `.ttc`, `.woff`, and `.woff2`), apart from `_public/` and dot-prefixed paths, is loaded into one font database, whether or not `build` would copy it.
That lets a card use the exact files the site's CSS serves, such as `_assets/fonts/Recursive.woff2`.
WOFF and WOFF2 files are decoded to plain font data in Go before loading.
System fonts are never loaded, so a card renders the same on every machine.

Text picks a face by `font-family`, `font-weight`, and `font-style`, as in CSS.
If two files provide the same family, weight, and style, the one whose path sorts first wins.
Each `font-family` list in the SVG must name at least one loaded family; otherwise the output fails with an error naming the list and the loaded families.
Generic names like `sans-serif` count only if the site has a family by that name, which it normally won't, so a typo in a family name is an error rather than a silent fallback.

The database is built the first time a raster output or `measureText()` needs it, and rebuilt after a `ParseAll` that follows a change to a font file.

### `measureText()`

SVG has no text wrapping, and titles vary in length, so templates need to measure text.
A new built-in global measures a string with the same font database and shaping as the rasterizer:

```js
measureText("Building your own Ngrok", { family: "Recursive", size: 72, weight: 700 })
// the advance width in pixels, as a number
```

The second argument takes `family`, `size` (pixels, default `16`), `weight` (default `400`), `style` (default `normal`), and `letterSpacing` (pixels, default `0`).
An unknown family throws, like it does in the rasterizer.
`measureText` is available in every JSX file, not only output templates, since it has no side effects.

Wrapping is left to the site, because what counts as a good break (hyphens, widows, a maximum number of lines with an ellipsis) is a style choice. A word wrapper is a few lines in `_globals.jsx`:

```js
wrap = (text, font, width) => text.split(" ").reduce((lines, word) => {
  const last = lines[lines.length - 1];
  if (last && measureText(`${last} ${word}`, font) <= width) lines[lines.length - 1] = `${last} ${word}`;
  else lines.push(word);
  return lines;
}, []);
```

### Images in cards

`<image href="...">` may point at a file in the site by root-relative path (`/assets/avatar.png`), or be a `data:` URI.
Paths resolve against the site directory, after symlinks are resolved, and must stay inside it.
Remote URLs, `file:` URLs, relative paths, and paths outside the site fail the output with an error.
PNG, JPEG, GIF, WebP, and SVG images are supported.

### Errors

A raster output that returns something other than SVG, has a missing or oversized `width` or `height`, uses an unknown font family, or references a forbidden image fails the build, and in `serve` returns a 500 with the error, as page-outputs specifies for a template that throws.

### Example

The redesign's card, with the layout tags that point at it:

```jsx
// _outputs/card.png.jsx
const W = 1200, H = 630, title = { family: "Recursive", size: 72, weight: 800 };
const lines = wrap(page.title ?? site_title, title, W - 160).slice(0, 4);
<svg width={W} height={H} viewBox={`0 0 ${W} ${H}`}>
  <rect width={W} height={H} fill="#fff" />
  <text x="80" y="104" font-family="Silkscreen" font-size="32">PROGRIUM.XYZ</text>
  <text font-family="Recursive" font-size={title.size} font-weight={title.weight}>
    {lines.map((line, i) => <tspan x="80" y={232 + i * 84}>{line}</tspan>)}
  </text>
  <rect x="80" y="520" width="1040" height="4" fill="#000" />
  <text x="80" y="574" font-family="Recursive" font-size="28">{page.date ?? ""}</text>
</svg>
```

```jsx
// in _layout.jsx
{page.outputs?.card && <>
  <meta property="og:image" content={`${site_url}${page.outputs.card}`} />
  <meta property="og:image:width" content="1200" />
  <meta property="og:image:height" content="630" />
</>}
```

### Engine

The rasterizer is [resvg](https://github.com/linebender/resvg), a Rust SVG renderer with its own text layout and shaping, compiled to a WebAssembly module and run with [wazero](https://github.com/tetratelabs/wazero), a WebAssembly runtime written in pure Go.

- The module lives in the repository as source (a small Rust crate exposing `render` and `measure`, with a pinned `Cargo.lock` and toolchain) and as the built `.wasm` file, embedded in the binary with `go:embed`.
  A Makefile target rebuilds it, and CI rebuilds it and fails if the result differs from the committed file.
  Building taragen needs only Go; only changing the module needs Rust.
- The module gets no WASI filesystem, network, clock, or environment.
  Taragen passes the SVG, font data, and image data in as bytes and gets PNG bytes back.
- The module is compiled once per process, on first use, so sites without raster outputs pay nothing.
  Each render gets a fresh module instance with a memory limit, so a bad card can't leak state into the next.
- On amd64 and arm64 wazero compiles the module to native code. On linux-386 it falls back to its interpreter, which is slower but produces the same bytes.

## Rationale

Taragen needs a rasterizer that renders text with custom fonts, runs inside a Go binary without cgo, and works on every release platform. The options considered:

**resvg in WebAssembly under wazero (chosen).** resvg is a mature, widely used SVG renderer (it's the PNG step behind Vercel's `@vercel/og`) with full text support: font fallback lists, weights, kerning, shaping via rustybuzz, `tspan`, `letter-spacing`, and `text-anchor`. Run under wazero it needs no cgo and no system libraries, cross-compiles like the rest of taragen, and is sandboxed by construction. The costs are binary size (the module is a few megabytes, plus wazero), a one-time compile of the module on first use, and a Rust crate in the repository for anyone who needs to change it.

**An existing Go binding of resvg.** `kanrichan/resvg-go` already wraps resvg in WebAssembly for Go, with a font-loading API. It's GPL-3.0, and linking it would put taragen's MIT-licensed binary under the GPL, so taragen builds its own small module instead. resvg itself is permissively licensed.

**tdewolff/canvas, pure Go.** A capable vector library with text boxes, word wrapping, and PNG output. It doesn't read SVG, though, so the template API would be a taragen-specific drawing vocabulary rather than SVG, which authors can't preview in a browser or copy from elsewhere. It also lacks WOFF2 and shaping beyond kerning.

**oksvg and rasterx, pure Go.** These rasterize an SVG subset but don't render `<text>`. Taragen would have to lay out text and turn glyphs into paths itself, which is most of a text engine.

**HTML and CSS instead of SVG.** HTML would let the card reuse the page's CSS and get wrapping for free, which is why `@vercel/og` uses Satori to lay out JSX with flexbox before resvg draws it. No Go library implements CSS layout, though, and running Satori in goja means running yoga's layout engine in a JavaScript interpreter that can't run its WebAssembly build. SVG with `measureText` covers what cards need. An HTML subset could be layered on later by its own proposal.

**A headless browser (chromedp, or Playwright in CI).** A browser renders any HTML and CSS with real web fonts, and the gap discussion considered a CI step that screenshots a cards page with Playwright. That needs Node and Chromium in the deploy pipeline, gives no cards in `serve` or local builds, and breaks the single-binary promise. chromedp would need Chrome installed on every machine that builds. A site that wants HTML cards today can still do the Playwright step itself, pointing `og:image` at the files it writes.

**Return bytes from JavaScript.** page-outputs asks whether an output could return a `Uint8Array`. A JavaScript rasterizer in goja would be far too slow for hundreds of cards, and binary results would let any output write arbitrary bytes. Rasterizing in Go keeps outputs to strings and elements.

**Only the site's fonts, never system fonts.** A card that renders with the author's laptop fonts and different CI fonts is a card that changes on every deploy. Loading every font file in the site is the zero-configuration version: the files the CSS already uses just work. Failing on unknown families catches the typo that would otherwise silently fall back.

**`measureText` and not a built-in wrapper.** Measurement has to come from the engine so it agrees with the rendered PNG. Line breaking is style, and a JavaScript helper is short, so taragen provides the part only it can.

**SVG width and height set the size, not a fixed 1200 by 630.** Different sites and platforms want different sizes, and the SVG already says how big it is.

## Compatibility

- **Site conventions:** font files anywhere in the site become inputs to raster outputs. Sites without a PNG output are unaffected.
- **JSX runtime:** `measureText` becomes a built-in global. Built-ins are added after user globals, so a site that defines its own `measureText` would have it replaced.
- **Page data:** raster outputs read `scale` from their own `data()`. Page data is unchanged.
- **Build and serve:** outputs whose content type is `image/png` are now rasterized instead of written as text. Before this, page-outputs would have written such an output's SVG verbatim under a `.png` name, which no site wants.
- **Dependencies:** adds `github.com/tetratelabs/wazero` (Apache-2.0, no dependencies of its own) and an embedded resvg module. The binary grows by several megabytes.
- **Distribution:** unchanged platforms. Everything is still pure Go with `CGO_ENABLED=0`.

## Security

Card templates run in the same goja VM as other outputs and see the same globals plus `measureText`, which only reads loaded fonts.
The new input is SVG, which can reference external resources. The rasterizer reads none itself: it runs in a WebAssembly sandbox with no filesystem, network, or clock, and taragen resolves every `<image>` reference before rendering, allowing only `data:` URIs and files inside the site directory after symlinks are resolved.
Page data that flows into a card, such as a title, is escaped by the JSX serializer like any text, so it can't inject markup into the SVG.
Width and height are capped at 4096 and scale at 4, and each render's module memory is limited, so a template can't make taragen allocate without bound.
The embedded `.wasm` is a binary in the repository. It's built from source in the repository with a pinned toolchain and lock file, and CI fails if a rebuild doesn't match it byte for byte.

## Done criteria

- [ ] `TestCardPNG`: `_outputs/card.png.jsx` returning a 1200 by 630 `<svg>` element builds `about.card.png` and `index.card.png` as valid 1200 by 630 PNGs whose pixels match the drawn shapes.
- [ ] `TestCardSVGString`: a template returning SVG markup as a string renders the same PNG as the equivalent element.
- [ ] `TestCardScale`: `data({ scale: 2 })` produces a 2400 by 1260 PNG; `scale: 5` and `width="5000"` fail with errors.
- [ ] `TestCardFonts`: text in a test font loaded from `testdata/` as TTF and as WOFF2 renders identical glyph pixels, and system fonts are never used.
- [ ] `TestCardUnknownFont`: a `font-family` naming no loaded family fails the build with the family named in the error.
- [ ] `TestCardImages`: `<image>` with a root-relative site path and with a `data:` URI renders; `https:`, `file:`, relative, `..`, and symlinked-out paths fail.
- [ ] `TestCardDeterministic`: two builds produce byte-identical PNGs.
- [ ] `TestMeasureText`: widths match the rendered text's extent within a pixel, weight changes the width, and an unknown family throws.
- [ ] `TestServeCard`: `serve` answers `/about.card.png` with `Content-Type: image/png`, and editing a font file or the template changes the next response.
- [ ] `TestCardNoEngine`: building a site with no raster output never compiles the module.
- [ ] CI rebuilds the resvg module from source and it matches the committed file.
- [ ] `make dist` builds for every release target with `CGO_ENABLED=0`, and a linux-386 build renders `TestCardPNG` correctly.
- [ ] The `web/` site adds `_outputs/card.png.jsx`, a font under `_assets/`, and `og:image` tags, and `make web-build` writes the cards.
- [ ] The spec describes raster outputs, font loading, `measureText`, and the new dependency.

## Implementation

1. Add the Rust crate for the resvg module (`render` and `measure` over a byte-buffer ABI), the Makefile target that builds it, the committed `.wasm`, and the CI reproducibility check.
2. Add a Go package that embeds the module, compiles it lazily with wazero, and exposes `Render(svg, fonts, images, scale)` and `Measure(text, font)`, with memory limits and the deterministic PNG settings.
3. Load site fonts into a database, decoding WOFF and WOFF2 in Go, and rebuild it after `ParseAll` when font files change.
4. Validate the SVG before rendering with `encoding/xml`: root size, font families, and image references, which are resolved to bytes.
5. Wire raster outputs into the page-outputs render path for `build` and `serve`, with `scale` from the output's data.
6. Add the `measureText` global.
7. Update the `web/` example site and the spec.

## Open issues

- Recursive is a variable font, and the redesign may use its `CASL` and `MONO` axes as well as weight. Confirm which axes the pinned resvg version honors through `font-variation-settings` or weight matching; if it doesn't, the site can ship static Recursive instances for cards, which the Recursive project publishes.
- Silkscreen is a pixel font and looks best without antialiasing at sizes on its pixel grid. Check that `text-rendering="optimizeSpeed"` gives crisp glyphs in resvg, or document the sizes to use.
- Should `serve` expose the SVG a card was rendered from, for example at `/about.card.png?svg`, to make debugging a layout faster?
- Builds now start clean, so every card is rasterized on every build. If that's slow for a large site, a cache keyed by a hash of the SVG and fonts could live outside `_public/`. Measure before adding it.
- JPEG or WebP outputs would reuse everything here except the encoder. Left out until a site needs one.
