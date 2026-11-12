---
slug: page-outputs
title: Extra output files per page
status: draft
created: 2026-11-12
authors: [progrium]
depends: []
areas: [site, data, jsx, serve]
branch: []
---

# Proposal: Extra output files per page

Nov 12, 2026 · Jeff Lindsay, h.d.a.

## Abstract

Let a site write extra files for each page, such as a plain text version at `/articles/2021/ngrok.txt` next to `/articles/2021/ngrok/`.
Output templates live in `_outputs/` folders and apply to every page in their folder and below, with the nearest one winning, the way `_layout` files do.
Each page gets a `page.outputs` map so layouts and listings can link to its extra files.
`build` and `serve` share one route table, so outputs are written by `build` and rendered on demand by `serve`.

## Background

Taragen writes exactly one file per page source (see Build mode in the [spec](../spec.md)).
A page without an extension goes to `<path>/index.html`, and a page whose name keeps an extension, like `feed.xml.jsx`, goes to `feed.xml`.
There is no way for one source to produce a second file.

The progrium.xyz redesign needs two files beyond the HTML for every page: a plain text version linked from each page's footer and `rel="alternate"` tag, and a share card image for `og:image`.
Writing a second source file per page by hand would double the content tree and drift out of sync.

Redirects from old URLs (page `aliases`) are the same shape: extra files written because a page exists.
So are generated pages (month archives, tag pages without a source file), except that those have no page behind them.

Today `build` walks the source tree and `serve` checks `Site.pages` and then the disk.
The two modes decide independently which URL maps to which file, which is part of why a directory and its `index` file render twice in `build`.

## Proposal

### Output templates

An output template is a file `_outputs/<name>.jsx`, in any directory of the site.
`<name>` is the output's name, and the rest of the file name after the first `.` is its URL suffix:

| File | Name | Suffix |
| --- | --- | --- |
| `_outputs/txt.jsx` | `txt` | `.txt` |
| `_outputs/card.png.jsx` | `card` | `.card.png` |

A name is a lowercase word of letters and digits, starting with a letter.
The suffix is the name plus any further extensions in the file name, so it is known from the file name alone, without running the template.

Output templates are scoped like layouts.
A template applies to pages in its directory and below.
For each name, the template in the nearest directory wins, so `logs/_outputs/card.png.jsx` replaces `_outputs/card.png.jsx` for pages under `logs/`.
A nearer template with the same name but a different suffix replaces the farther one, and the page gets the nearer suffix.

`_outputs/` starts with `_`, so the walker already keeps it out of the page tree.

### Which pages get outputs

A page gets outputs when it has a source and its `contentType` is `text/html`.
Bare directories without an index get none, and neither do pages that set another content type, such as `feed.xml.jsx`.

A page turns outputs off in its data, most often in frontmatter or `data()`:

```yaml
outputs: { card: false }
```

`outputs: false` turns off every output for that page.
Entries that name an output that doesn't apply to the page are ignored.
Because this is ordinary page data, a `_defaults.jsx` can turn an output off for a whole directory.

A draft's outputs are drafts too: `build` skips them, and `serve` (which shows drafts) serves them.

### URLs

An output's URL is the page's URL with the suffix appended, and the root page uses `index`:

| Page | `txt` | `card` |
| --- | --- | --- |
| `/articles/2021/ngrok` | `/articles/2021/ngrok.txt` | `/articles/2021/ngrok.card.png` |
| `/` | `/index.txt` | `/index.card.png` |

`build` writes each output to the same path under `_public/`, so `_public/articles/2021/ngrok.txt` sits beside `_public/articles/2021/ngrok/index.html`.

### Rendering an output

An output renders after its page has been parsed.
The template runs in a fresh VM with the page's globals, like a layout, and `page` is the page's data, including `src` and `body` (the page body before layouts).
The page's layouts are not applied, and the template's completion value is the whole file.
If the completion value is a function, it is called with no arguments.

The result may be:

- a string, written verbatim with no escaping or formatting, or
- an element, serialized to HTML as for pages.

Output templates are new in returning a bare string; pages, layouts, and partials keep their current rules.

Inside an output template, `data()` sets the output's own data, not the page's.
The keys it reads are:

| Key | Meaning |
| --- | --- |
| `contentType` | The `Content-Type` in `serve`. Defaults to the type for the last extension of the suffix (`text/plain; charset=utf-8` for `.txt`), or `application/octet-stream` if there isn't one. |
| `preamble` | Text written before the output, as for pages. Defaults to none, including for `text/html`. |

An output with `contentType: text/html` is formatted with `gohtml` like a page.
A template that throws fails the build, and in `serve` returns a 500 with the error, as pages do.

Example, the plain text version used by the redesign:

```jsx
// _outputs/txt.jsx
data({ contentType: "text/plain; charset=utf-8" });
`${page.title ?? page.date}\n\n${page.src}`
```

### `page.outputs`

Before any page is parsed, taragen sets `outputs` in each page's data to a map from output name to URL, for the outputs that apply to it and are not turned off:

```js
page.outputs // { txt: "/articles/2021/ngrok.txt", card: "/articles/2021/ngrok.card.png" }
```

The map replaces whatever `outputs` value the page set, after the opt-outs are read.
Since the URLs come from file names, the map is complete during parsing, so layouts can use it for `<link rel="alternate">` and `og:image`, and `pages()` results carry it for feeds and listings.

### Route table

`ParseAll` builds a route table that maps every URL the site serves to what produces it: a page, an output of a page, or a static file.
Each URL appears once.
Two producers claiming the same URL is an error naming both, for example an output `foo.txt` and a static file `foo.txt`.
Drafts are kept in the table and marked, so `build` and `serve` can each apply their draft rule.

`build` walks the route table instead of the source tree: it renders pages and outputs and copies static files.
`serve` looks the request path up in the table and renders the page or output on demand, injecting the live reload script only into `text/html` responses.
Paths not in the table fall through to the current `http.ServeFile` behavior.

Adding or changing a file under `_outputs/` is a `.jsx` change, so the watcher's existing `ParseAll` picks it up.

### Out of scope

These build on this proposal and get their own proposals:

- **Aliases:** a built-in output that writes one redirect stub per entry in `page.aliases`. Its URLs come from page data, not a suffix, so it needs a way to add rows to the route table beyond the suffix rule.
- **Share cards:** an output with `contentType: image/png` whose template returns SVG or HTML for taragen to rasterize. The rasterizer is the separate part; the wiring is this proposal.
- **Generated pages:** many pages from one template with no source file, such as month archives. They would add rows to the same route table.

## Rationale

**`foo.txt` beside `foo/`, not `foo/index.txt`.** `foo.txt` is the URL people type and `curl`, and it reads as "this page, as text". `foo/index.txt` would avoid placing a file next to a directory, but common static hosts (GitHub Pages, Netlify, S3) serve `foo.txt` and `foo/index.html` side by side without trouble, and `foo/index.txt` is not reachable at a clean URL anyway. This was the choice made when the design was discussed.

**Suffix from the file name, not from `data()`.** An earlier sketch had the template set `ext` (and optionally `path`) with `data()`. That means running every template for every page before any URL is known, and pages can't see `page.outputs` while they are being parsed. Taking the suffix from the file name keeps the URLs static, matches the existing `feed.xml.jsx` convention, and still lets `data()` set the content type. A per-page path override is left to the aliases proposal, which needs it anyway.

**Scoped like layouts.** The filesystem is taragen's configuration, and nearest-wins scoping is already how layouts, globals, and defaults work. A site-wide config list of outputs would be the first config file.

**A folder, not `_txt.jsx` files.** Any other `_name.jsx` is already a partial, so a reserved prefix would make output templates ambiguous with partials. A `_outputs/` folder is unambiguous and lists a directory's outputs in one place.

**Only HTML pages get outputs.** A `.txt` of `feed.xml` or a card for a JSON index is never wanted, and excluding them saves every such page an opt-out.

**One route table for both modes.** Without it, `build` and `serve` would each need their own output logic, and they already disagree on underscore paths. The table also removes the double render of a directory and its `index` file, since each URL appears once.

**Opt-out reuses the `outputs` key.** One key for both the switch and the result keeps the data model small. The cost is that a page reading `page.outputs` never sees the `false` it set, which is the point of turning an output off.

## Compatibility

- **Site conventions:** `_outputs/` becomes a special directory. Sites that already have an `_outputs/` directory of partials would see those files become output templates. No known site does.
- **Page data:** `outputs` becomes a key with built-in meaning, and taragen overwrites it. A site already using `outputs` for its own data would lose that value.
- **JSX runtime:** output templates may return a bare string. Nothing else changes for pages, layouts, or partials.
- **Build:** sites without `_outputs/` build the same files as before, except that a directory and its `index` file render once instead of twice. A URL collision that used to silently overwrite a file in `_public/` now fails the build.
- **Serve:** output URLs are served. Other requests behave as before.
- **CLI and distribution:** unchanged.

## Security

Output templates run in the same sandboxed goja VM as every other JSX file and see the same globals, so nothing new becomes reachable to site code.
Output URLs are derived from page paths and template file names, both of which are already inside the site root, and collisions are errors, so an output can't write outside `_public/` or overwrite another file.

## Done criteria

- [ ] `TestOutputTxt`: a site with `_outputs/txt.jsx` builds `about.txt` and `index.txt` with the template's string output, verbatim.
- [ ] `TestOutputSuffix`: `_outputs/card.png.jsx` produces `about.card.png`, keyed `card` in `page.outputs`.
- [ ] `TestOutputNearestWins`: `logs/_outputs/txt.jsx` overrides the root template for pages under `logs/` only.
- [ ] `TestOutputOptOut`: `outputs: { txt: false }` and `outputs: false` in frontmatter, and in a `_defaults.jsx`, remove the files and the map entries.
- [ ] `TestOutputSkips`: drafts, bare directories, and non-HTML pages like `feed.xml.jsx` get no outputs in `build`; drafts' outputs are served by `serve`.
- [ ] `TestOutputsMap`: `page.outputs` is visible to the page's own layout during render and in `pages()` results from another page.
- [ ] `TestOutputContentType`: the default content type comes from the suffix, `data({contentType})` overrides it, and `serve` sends it.
- [ ] `TestOutputDataIsolated`: `data()` in an output template doesn't change the page's data.
- [ ] `TestRouteCollision`: an output URL that matches a static file or another page fails `build` with both producers named.
- [ ] `TestRouteSingleRender`: a directory with an `index.jsx` renders once in `build`.
- [ ] `TestServeOutput`: `serve` renders `/about.txt` on request, injects no live reload script into it, and reflects template edits after a reload.
- [ ] The `web/` site adds `_outputs/txt.jsx` and a footer link to `page.outputs.txt`, and `make web-build` writes the `.txt` files.
- [ ] The spec describes output templates, `page.outputs`, and the route table.

## Implementation

1. Add the route table to `ParseAll`, covering pages and static files only, and switch `build` and `ServeHTTP` to it. This fixes the double render and changes no output.
2. Find `_outputs/` templates per page with nearest-wins scoping, read opt-outs, and set `page.outputs` before pages are parsed.
3. Add output rows to the route table, with collision errors.
4. Render outputs: a fresh VM with the page's globals, output-scoped `data()`, and string results.
5. Serve outputs from the table, with content types and no live reload injection for non-HTML.
6. Update the `web/` example site and the spec.

## Open issues

- Should output templates also be allowed as `.tmpl`, like layouts? JSX covers the redesign, so this proposal leaves it out.
- Should an output be able to return bytes (a `Uint8Array`) for binary formats? Share cards would rasterize in Go instead, so this is deferred to that proposal.
