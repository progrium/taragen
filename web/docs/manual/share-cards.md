---
title: "Share cards"
kind: manual
num: 18
order: 18
part: "outputs"
toc:
  - ["a-card-template", "A card template"]
  - ["rules-for-card-svg", "Rules for card SVG"]
  - ["fonts", "Fonts"]
  - ["measuring-text-with-measuretext", "Measuring text with `measureText()`"]
  - ["images-in-cards", "Images in cards"]
  - ["card-errors", "Card errors"]
---

A share card is the image social sites show for a link (`og:image`). In taragen
it's an ordinary output whose content type is `image/png`: the template returns
SVG, and taragen rasterizes it into a PNG using the site's own fonts. No
headless browser or external tool is involved.

<a id="a-card-template"></a>

## A card template

Name the template `_outputs/<name>.png.jsx` and return an `<svg>` with numeric
`width` and `height`:

```jsx
// _outputs/card.png.jsx
const W = 1200, H = 630, title = { family: "Recursive", size: 72, weight: 800 };
const lines = wrap(page.title ?? siteTitle, title, W - 160).slice(0, 4);
<svg width={W} height={H} viewBox={`0 0 ${W} ${H}`}>
  <rect width={W} height={H} fill="#fff" />
  <text x="80" y="104" font-family="Silkscreen" font-size="32">EXAMPLE.COM</text>
  <text font-family="Recursive" font-size={title.size} font-weight={title.weight}>
    {lines.map((line, i) => <tspan x="80" y={232 + i * 84}>{line}</tspan>)}
  </text>
  <rect x="80" y="520" width="1040" height="4" fill="#000" />
  <text x="80" y="574" font-family="Recursive" font-size="28">{page.date ?? ""}</text>
</svg>
```

(`wrap` is the helper shown in [Defaults and globals](/docs/manual/defaults-and-globals).)

Point your layout at it. `og:image` needs an absolute URL, so use `siteURL`:

```jsx
// in _layout.jsx
{page.outputs?.card && <>
  <meta property="og:image" content={`${page.siteURL}${page.outputs.card}`} />
  <meta property="og:image:width" content="1200" />
  <meta property="og:image:height" content="630" />
</>}
```

`build` writes `/articles/2021/ngrok.card.png` for every HTML page, and `serve`
renders each card when it's requested, so you can open a card URL in the browser
while designing it.

<a id="rules-for-card-svg"></a>

## Rules for card SVG

- Return a root `<svg>` element, or a string of SVG markup. The `xmlns`
  attribute is added if it's missing.
- The root needs numeric `width` and `height` in pixels, each at most 4096.
  These set the image size.
- `data({ scale: 2 })` renders at 2× (any whole number from 1 to 4) without
  changing your coordinates, for sharper cards on dense screens.
- Supported: shapes, paths, gradients, patterns, clipping, masks, filters,
  `<text>`, `<tspan>`, and `<image>`. Scripts, animation, and
  `<foreignObject>` are ignored. SVG has no text wrapping; measure and break
  lines yourself (below).
- Any output can be made a PNG output with
  `data({ contentType: "image/png" })`, whatever its suffix.

The same SVG and fonts always produce byte-identical PNGs.

<a id="fonts"></a>

## Fonts

Cards can use only fonts that are in your site. Every `.ttf`, `.otf`, `.ttc`,
`.woff`, and `.woff2` file anywhere in the site folder (except `_public/` and
dot-prefixed paths) is available, so a card can use the same font files your
CSS serves, such as `_assets/fonts/Recursive.woff2`. System fonts are never
used, so cards look the same on every machine.

Text picks a face by `font-family`, `font-weight`, and `font-style`, as in CSS.
If two files provide the same family, weight, and style, the one whose path sorts
first wins.

Every `font-family` must name at least one font your site has. Generic names
like `sans-serif` don't count unless you have a font with that name. An unknown
family fails the card with an error that lists the families taragen found, so a
typo shows up immediately instead of silently falling back.

Variable fonts work. `font-weight` sets the `wght` axis (up to 900; for heavier
weights use `font-variation-settings="'wght' 1000"`), and
`font-variation-settings` sets any other axis, such as Recursive's `CASL` and
`MONO`. A font with a `slnt` axis but no `ital` axis, like Recursive, ignores
`font-style="italic"`; use `font-style="oblique"` or `'slnt' -15` instead.

Pixel fonts stay crisp at sizes on their pixel grid with whole-pixel `x` and
`y`. Silkscreen, for example, is 8 pixels per em, so use 16, 24, 32, and so on.
`text-rendering="optimizeSpeed"` turns antialiasing off at any size.

<a id="measuring-text-with-measuretext"></a>

## Measuring text with `measureText()`

`measureText(text, font)` returns the width of a string in pixels, using the
same fonts and shaping as the rasterizer:

```js
measureText("Building your own Ngrok", { family: "Recursive", size: 72, weight: 700 })
```

<div class="tw"><table>
<thead><tr><th>Option</th><th>Default</th></tr></thead>
<tbody>
<tr><td><code>family</code></td><td class="prose">required</td></tr>
<tr><td><code>size</code></td><td class="prose"><code>16</code> (pixels)</td></tr>
<tr><td><code>weight</code></td><td class="prose"><code>400</code></td></tr>
<tr><td><code>style</code></td><td class="prose"><code>&#34;normal&#34;</code></td></tr>
<tr><td><code>letterSpacing</code></td><td class="prose"><code>0</code> (pixels, added between letters, not after the last)</td></tr>
</tbody>
</table></div>

An unknown family throws. `measureText` works in any JSX file, not just card
templates. Line breaking is up to you; the `wrap` helper above is a simple word
wrapper you can extend with ellipses or line limits.

<a id="images-in-cards"></a>

## Images in cards

`<image href="...">` can use a root-relative path to a file in your site
(`/_assets/avatar.png`) or a `data:` URI. PNG, JPEG, GIF, WebP, and SVG are
supported. Remote URLs, `file:` URLs, relative paths, and paths that lead
outside the site folder are errors.

<a id="card-errors"></a>

## Card errors

A card fails the build (or returns a 500 in `serve`) when the template doesn't
return SVG, `width` or `height` is missing or over 4096, a font family is
unknown, or an image reference isn't allowed.
