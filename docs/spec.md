# Taragen spec

This spec describes taragen as built: its architecture, the main components, how
a build or request flows through them, and the decisions (and known rough edges)
baked into the current code. It describes what exists, never unbuilt features.
Changes to it go through proposals; see [the design process](design/README.md).
Its starting point is frozen in the [baseline](design/baseline.md) proposal.

## Overview

Taragen is a static site generator distributed as a single Go binary. A site is
a directory of files. Page sources can be **JSX**, **Markdown**, or **Go
templates**; everything else is copied as a static file. JSX is compiled with
esbuild and executed in an embedded JavaScript VM (goja), so there is no Node.js
dependency and no `node_modules`.

The binary has two modes:

- `taragen serve` parses the site, serves it on `http://localhost:8088`
  (including drafts), renders pages on demand, and live reloads the browser when
  sources change.
- `taragen build` (aliases `gen`, `generate`) cleans `_public/` in the current
  directory and renders every page to static files there.

Both commands operate on the current working directory as the site root.

## Repository layout

| Path | Role |
| --- | --- |
| `cmd/taragen/` | CLI entry point (`main.go`, `serve.go`, `build.go`) built on `tractor.dev/toolkit-go` engine/cli. |
| `site.go` | `Site`: directory walking, page registry, static build, HTTP handler, partial loading. |
| `routes.go` | `RouteTable`: every URL the site serves and the page, output, static file, alias, or redirect that produces it. |
| `page.go` | `Page`: page data model, parsing, globals/defaults loading, layout chain, rendering. |
| `outputs.go` | `Output`: `_outputs/` template lookup, opt-outs, `page.outputs`, and output rendering. |
| `aliases.go` | Reading and normalizing `aliases`, and writing redirect stubs. |
| `redirects.go` | Parsing `_redirects`, adding its route rows, and matching wildcard rules in serve. |
| `format.go` | Format registry (`Formats`), `PageParser` interface, YAML frontmatter splitting. |
| `format_jsx.go` | JSX page parser and the built-in JS globals (`page`, `partial`, `pages`). |
| `query.go` | `pages()` queries: pattern matching with `**`, the object form with `where`, and the listing order. |
| `format_md.go` | Markdown parser (template pass, then goldmark with syntax highlighting). |
| `format_tmpl.go` | Go `text/template` parser, template helper funcs, `RenderTemplate`. |
| `jsx.go` | JSX runtime: esbuild transform, goja VM setup, `hyper` element factory, HTML serialization. |
| `reloader.go` | File watcher and WebSocket-based live reload. |
| `*_test.go` | Table tests for JSX rendering, and page, output, build, serve, and watcher tests on temporary sites. |
| `web/` | The taragen website itself, which doubles as the reference example site. |
| `Makefile`, `.github/workflows/build.yml` | Local builds, cross-compiled dist binaries, CI via a shared reusable workflow. |
| `tools/design/`, `internal/design/` | The design process tool: proposal parsing, `check`, computed state, and signing. |
| `docs/spec.md`, `docs/design/` | This spec, and the design proposals that change it. |
| `AGENTS.md` | Rules for AI agents working in the repository. |

## Core concepts

### Site and file conventions

A `Site` is rooted at a directory. `Site.walk` traverses it with these rules:

- Names starting with `.` are skipped (directories are pruned).
- Names starting with `_` are skipped (directories are pruned). This is how
  layouts, partials, globals, defaults, output templates, and assets stay out
  of the page tree.
  It also means the build output directory `_public/` is never re-ingested.
  `build` copies the root `_assets/` directory separately (see Build mode).
- A file whose name ends in a registered format extension (`.md`, `.jsx`,
  `.tmpl`) is a page source. Directories are also visited as potential pages.
- Every other file is a static file.

Special files, all resolved relative to the site root:

| File | Purpose |
| --- | --- |
| `_layout.jsx` / `_layout.tmpl` | Directory layout wrapping every page in that directory and below. |
| `_globals.jsx` | JS values and helper functions available to all JSX in that directory and below. |
| `_defaults.jsx` | Default page data for pages in that directory and below. |
| `_outputs/<name>.jsx` | An output template: an extra file for every HTML page in that directory and below (see Outputs). |
| any other `_name.jsx` / `_name.tmpl` | A partial, loaded by path via `partial("_dir/name")`. |
| `_assets/` (root only) | Static files copied verbatim to `_public/_assets/` by `build`, never rendered. |
| `_redirects` (root only) | Redirect rules in a subset of Netlify's format, copied to `_public/_redirects` by `build` (see Aliases and redirects). |

### Page paths

Source paths are normalized into page paths by `Site.normalizePath`: strip the
site root, strip the extension, strip a trailing `/index`, and use `.` for the
root. So `about.jsx` and `about/index.jsx` both become page `about`, and
`index.jsx` becomes `.`.

Pages live in `Site.pages`, a `map[string]*Page` keyed by normalized path.
`Site.Page(path)` lazily creates an entry with `path` and `slug` seeded into its
data. Directories without an index file still get a `Page` (with `isDir: true`
and no source) so they can act as grouping nodes for `pages()` queries. A page
is never in its own `subpages`, including the root.

### Page data

Each `Page` carries a `Data` map (`map[string]any`). It is the single bag of
state shared between Go and JS and exposed to templates as `page`. Keys with
built-in meaning:

| Key | Meaning |
| --- | --- |
| `path` | URL path, e.g. `/docs/getting-started` (seeded). |
| `slug` | Last path segment (seeded). |
| `src` | Raw page source. |
| `body` | Rendered page body before layouts. |
| `isDir` | True when the page is a directory or an `index.*` file. |
| `layout` | Explicit layout path (e.g. `_blog/_layout`); `""` disables layouts. |
| `date` | `YYYY-MM-DD`; used for sorting and future-dated drafts. |
| `draft` | Explicit draft flag. |
| `contentType` | Defaults to `text/html`; served as the HTTP `Content-Type`. |
| `preamble` | Text written before the output; defaults to `<!DOCTYPE html>` for HTML. |
| `outputs` | Set by taragen to a map from output name to URL (see Outputs). A page sets it to `false`, or to a map with `false` entries, to turn outputs off. |
| `aliases` | Old URL paths that redirect to the page, as a string or a list of strings. Rewritten by taragen to the normalized list (see Aliases and redirects). |

Any other keys (`title`, `description`, ...) are free-form and flow through to
JSX and templates untouched.

Data is populated in this order, later writers winning:

1. Seeded `path` and `slug`.
2. `_defaults.jsx` files from the root down to the page's directory.
3. Format-specific data: YAML frontmatter for Markdown and templates, or calls to
   `data({...})` in JSX.
4. Frontmatter from Go template layouts, and `data()` calls in JSX layouts,
   during rendering.

### Drafts

`Page.Draft()` is true if `draft: true` is set **or** the page's `date` is in
the future. A draft's outputs and aliases are drafts too. `build` always skips drafts. `serve` constructs the site with
`drafts=true`, so drafts are still rendered and included in `pages()` listings.
`Subpages()` filters drafts only when the site is not in drafts mode.

## Formats

Formats are registered in a global map keyed by file extension:

```go
var Formats = map[string]PageParser{
	".md":   &MarkdownParser{},
	".tmpl": &TemplateParser{},
	".jsx":  &JSXParser{},
}

type PageParser interface {
	Parse(p *Page) (body []byte, data Data, err error)
}
```

A parser turns a page's source into a body and returns any data it discovered.
Adding a format means implementing `PageParser` and adding it to the map; the
walker, page resolution, and watcher pick it up by extension.

### JSX

JSX is the primary format and the most involved. The pipeline in `jsx.go`:

1. **Pre-processing.** `replacePreNewlines` rewrites newlines inside `<pre>`
   blocks to `<newline />` elements so they survive JSX whitespace collapsing.
2. **Transform.** esbuild transforms the source with the JSX loader, using
   `hyper` as the factory and the string `'<>'` as the fragment, so
   `<div class="x">hi</div>` becomes `hyper("div", {class: "x"}, "hi")`.
3. **Execute.** A fresh goja VM is created for every render. `hyper` and `raw`
   (Go functions bound to that VM) and all globals are set on it, and the
   transformed code is run.
4. **Call.** If the script's completion value is a function, it is called with
   the render arguments. This is how layouts (`(content) => ...`) and partials
   with parameters work; plain pages evaluate straight to an element.
5. **Serialize.** The result is converted to a Go `hyperNode` tree and
   serialized to HTML, then `<newline></newline>` markers are turned back into
   real newlines. A marker that landed inside a JS string (in a `<pre>`) is
   turned back into a newline in the text node.

`hyper` accepts children that are other nodes, strings, numbers, or arrays
(flattened, so `.map(...)` works); `null`, `undefined`, and booleans render
nothing, so `{cond && <b/>}` works. Numbers are formatted as JS prints them.
Attribute values may be strings (empty strings are omitted), numbers, booleans
(emitted as bare attributes when true, omitted when false), or objects
(serialized as a `style` string); `null` and `undefined` omit the attribute.
Void elements such as `img`, `meta`, and `br` are self-closed when they have no
children. Anything else throws a JS `TypeError`, which fails the render.

**Attribute order.** `hyper` reads attributes from the JS object's own key
order, so they are written in source order, as are the keys of a style object.
Output is the same on every build.

**Escaping.** Text children are escaped (`&`, `<`, `>`) and attribute values are
escaped (`&`, `<`, `>`, `"`, `'`). Text inside `<script>` and `<style>` is
written as is. `raw(html)` returns a node whose string is inserted unescaped
(`null`/`undefined` give an empty node). Partials and a JSX layout's `content`
argument are raw nodes, because they are already rendered HTML; any other HTML
string, such as `post.body` from `pages()`, needs `raw()`.

Output is a string, not a DOM: there is no client-side runtime, no hydration, and
no component state. JSX is purely a templating language here.

**Page wrapping.** `JSXParser` wraps page sources in a `<>...</>` fragment so a
page can have multiple root elements or start with a bare `{partial(...)}`
expression. If the first statement (after whitespace and comments) is a
`data(...)` call, it is kept outside the fragment. Its end is found by matching
parentheses while skipping strings, template literals, and comments, and a
trailing `;` is optional. Pages that mention `data` anywhere else are wrapped
whole.

**`data()`.** When the globals include `page`, `RenderJSX` installs a `data(obj)`
function that merges `obj` into both the JS `page` object and the Go page `Data`
map. This is how JSX pages set `title`, `layout`, `contentType`, and so on.
In a page body, a `data()` call that sets `outputs` is read as an opt-out right
away, so `page.outputs` stays the map of URLs. Output templates get their own
`data()` (see Outputs).

**Globals.** `ExportJSX` runs a `_globals.jsx` or `_defaults.jsx` file as a
script and returns every global key that did not exist before execution. That is
why those files use bare assignments (`siteTitle = "Taragen"`,
`toTitleCase = (s) => ...`) rather than `export`.

Built-in globals, added after user globals so they always win:

- `page`: the current page's `Data` map.
- `partial(name, ...args)`: renders `_name.jsx` (falling back to `_name.tmpl`)
  with the same globals and returns the HTML as a raw node. A missing or failing
  partial throws an error naming it, which fails the page render.
- `raw(html)`: inserts an HTML string unescaped (see Escaping above). It is set
  in every JSX VM, including layouts and partials.
- `pages(query)`: returns page data objects (each with a recursive `subpages`
  array), sorted as one listing (see Ordering). See Page queries below.

### Page queries

`pages()` takes a string or an object. Both are implemented in `query.go`.

A string is a path or a pattern:

- A plain path, such as `pages("docs")`, returns the subpages of that page.
- A string containing `*` is a pattern. It returns the subpages of every page
  whose path matches it; put another way, the pages whose parent matches. It is
  matched one `/`-separated segment at a time with `path.Match` rules, so `*`,
  `?`, and `[...]` never cross a `/`. The root's path is empty for matching, so
  `pages("*")` returns the pages two levels deep, not the root's subpages.
- A segment that is exactly `**` matches zero or more whole segments:
  `pages("articles/**")` returns every page below `articles` at any depth,
  `pages("**")` every page but the root, and `docs/**/api` matches at any
  depth. `**` inside a longer segment (`a**`) is just two `*`s. Grouping nodes,
  pages without a source, are left out of the results of a pattern with a `**`
  segment; plain paths, `*` patterns, and `subpages` still return them.

The object form is `pages({ path, where })`:

| Key | Default | Meaning |
| --- | --- | --- |
| `path` | `"**"` | A path or pattern as above, or a list of them. A list returns the union of each one's result, once each, sorted as one listing. |
| `where` | none | Page data keys to wanted values; a page is kept only when every key matches. |

A `where` key matches when the wanted string, number, or boolean is strictly
equal (as `===`) to the page's value or to an element of the page's list value,
or, for a wanted list, when any element of it does. A page without the key never
matches, and `where` never filters `subpages`.

A page appears at most once in a result. A malformed pattern, an unknown key, a
`path` that isn't a string or a list of strings, a `where` value that is an
object or `null`, or an argument that is neither a string nor an object throws
an error naming the query, which fails the render.

### Markdown

Markdown first goes through the Go template parser (so frontmatter is extracted
and `[[ ... ]]` template actions run), then through goldmark with the
`goldmark-highlighting` (Chroma) extension. Raw HTML is allowed
(`html.WithUnsafe()`).

### Go templates

`.tmpl` files use `text/template` with `[[` and `]]` as delimiters. The custom
delimiters avoid clashing with `{...}` and `{{...}}` that are common in
content and code samples, and `text/template` (rather than `html/template`) is
used so the output is not auto-escaped. Frontmatter is split off first.

Template helper functions:

- `content`: the wrapped content (when used as a layout) or page body.
- `page "key"`: a string value from page data.
- `partial "name" args...`: same partial lookup as JSX.
- `safeHTML`: identity, kept for compatibility.

### Frontmatter

`SplitFrontmatter` looks for `---\n` delimiters and parses the first block as
YAML (`gopkg.in/yaml.v2`). If the source has no `---\n` at all, it is treated as
having no frontmatter.

## Rendering flow

### Parsing a page

`Page.Parse()`:

1. Try each format extension as `<path><ext>`, then `<path>/index<ext>`. The
   first file found wins; an `index` hit sets `isDir`.
2. If no source is found but the path is a directory, mark it `isDir` and stop.
3. Load globals: collect `_globals.jsx` files from the root down to the page's
   directory, run each in order with the accumulated globals so far, then add the
   built-in globals.
4. Load defaults: collect `_defaults.jsx` the same way and merge their exports
   into page data.
5. Find the output templates that apply and set `page.outputs`, applying any
   opt-out the defaults set (see Outputs).
6. Store the source in `src`, run the format's parser, merge the returned data,
   and store the result in `body`.
7. Set `page.outputs` again from the page's final data: empty unless the
   page's `contentType` is `text/html`, and without the outputs it turned off.

### Rendering a page

`Page.Render(w)` re-parses the page (so it always reflects the latest source),
then applies layouts:

1. **Find default layouts.** Unless the page set `layout` explicitly, walk from
   the page's own path up to the root, collecting `_layout.jsx` and
   `_layout.tmpl` files nearest-first.
2. **Apply the chain.** Starting with the nearest layout, render it with the
   current output as its content. JSX layouts receive it as their single
   function argument, as a raw node; template layouts get it through `content`
   and may set data (including `layout`) in frontmatter.
3. **Explicit overrides.** If rendering a layout changes `page.layout`, the
   remaining default layouts are discarded and the newly named layout is
   applied next. This lets a page or layout opt into a specific chain. Setting
   `layout: ""` (as `feed.xml.jsx` does) disables layouts entirely.
4. **Output.** Write the preamble and a newline, then the body. For
   `text/html`, the output is passed through `gohtml` in condensed mode to
   normalize formatting.

The original `layout` value is restored afterwards so re-rendering is stable.

```
source file
   │  Page.Parse
   ▼
_globals.jsx chain ──► globals ─┐
_defaults.jsx chain ─► data ────┤
format parser (jsx/md/tmpl) ────┴─► body + data
   │  Page.Render
   ▼
nearest _layout ─► parent _layout ─► ... ─► root _layout
   │
   ▼
preamble + gohtml-formatted output
```

### Outputs

An output is an extra file written for a page, such as a plain text version at
`/about.txt` beside `/about/`. Each comes from an output template,
`_outputs/<name>.jsx`, in any directory of the site. `<name>` is a lowercase
letter followed by lowercase letters and digits, and the rest of the file name
after the first `.` makes the URL suffix: `_outputs/txt.jsx` is output `txt`
with suffix `.txt`, and `_outputs/card.png.jsx` is output `card` with suffix
`.card.png`. Other names fail the parse, as do two templates in one directory
with the same name. Dot-prefixed and non-`.jsx` files in `_outputs/` are
ignored.

Templates are scoped like layouts: looking from the page's own path up to the
root, the nearest template wins for each name, and a nearer template replaces
a farther one even when its suffix differs. Only pages with a source and a
`contentType` of `text/html` get outputs; bare directories and pages like
`feed.xml.jsx` get none.

**Opt-outs.** `outputs: false` in page data turns every output off for that
page, and `outputs: { card: false }` turns off the named ones. Names that don't
apply are ignored. It is ordinary page data, so it can come from frontmatter,
`data()`, or a `_defaults.jsx` (`outputs = {card: false}`), with the usual
precedence: the page's own value replaces the defaults' whole.

**`page.outputs`.** Parsing sets `outputs` in page data to a map from output
name to URL for the outputs that are on, replacing the opt-out value. It is set
before the body runs, so the body, its layouts, and `pages()` results from
other pages can all link to `page.outputs.txt`. Every parsed page has the key,
as an empty map when it has no outputs.

**URLs.** An output's URL is the page's URL with the suffix appended, using
`/index` for the root: `/articles/ngrok.txt` for page `/articles/ngrok`, and
`/index.txt` for `/`. `build` writes it to the same path under `_public/`.

**Rendering.** `Output.Render` re-parses the page, then runs the template in a
fresh VM with the page's globals, so `page` is the page's data including `src`
and `body` (the body before layouts). Layouts are not applied. The completion
value is the whole file; a function is called with no arguments first. A string
result is written verbatim, without escaping or formatting, and an element is
serialized as for pages. Inside the template, `data()` sets the output's own
data and never the page's. It reads:

| Key | Meaning |
| --- | --- |
| `contentType` | Defaults to the type for the suffix's last extension (`text/plain; charset=utf-8` for `.txt`), or `application/octet-stream`. |
| `preamble` | Written before the output, followed by a newline. Defaults to none, even for HTML. |

A `text/html` output is formatted with `gohtml` like a page. A template that
throws fails the build, or is a 500 in `serve`, with an error naming the output
and its page.

### Aliases and redirects

**Aliases.** A page lists the URLs it used to live at in `aliases`. It is
ordinary page data, so it can come from `_defaults.jsx`, frontmatter, or
`data()`, and a page's own value replaces the defaults' whole. It is read at
the end of `Parse`, so a value set by a layout is ignored. A single string is a
list of one; any other type, or a non-string entry, fails the parse naming the
page. Bare directories have no aliases.

Each alias, and each `from` in `_redirects`, is normalized: it must start with
`/`, a trailing `/` is removed (except for `/` itself), and it must not contain
`?`, `#`, `\`, an empty segment, or a `.` or `..` segment, so nothing can be
written outside `_public/`. Duplicates in one page's list are dropped, and
`page.aliases` is rewritten to the normalized list, which layouts and `pages()`
results see. A URL maps to a file like a page path: `<alias>/index.html` when
its last segment has no extension, and as-is otherwise.

**Stubs.** For an alias of an HTML page, `build` writes a fixed stub page with a
`<meta http-equiv="refresh">` to the page's path, a canonical link, a fallback
link, and `<meta name="robots" content="noindex">`. The target is escaped as an
attribute and as text, as in JSX output. Stubs run no layouts, globals, or
output templates. An alias of a non-HTML page, such as `feed.xml.jsx`, gets a
copy of the page's output instead, so feed readers that don't follow a refresh
keep working.

**The redirects file.** The root `_redirects` file holds one rule per line,
`from to [status]`; blank lines and lines starting with `#` are ignored.

- `from` is a normalized path that may end in `/*`, a wildcard matching every
  path below it. `*` anywhere else, and `:` placeholders, are errors.
- `to` is a path starting with `/` (with an optional query string) or an
  absolute `http` or `https` URL. It may contain `:splat`, replaced by what the
  wildcard matched, only when `from` ends in `/*`. Other placeholders, `#`, and
  `\` are errors.
- `status` is `301`, `302`, `307`, or `308`, defaulting to `301`.

Anything else (`200` rewrites, `404`s, the `!` force suffix, conditions, extra
fields) fails `ParseAll` naming the line. `build` copies the file verbatim and
writes stubs from it:

- A rule without a wildcard writes a stub at `from` pointing to `to`, or a copy
  of the page when `to` is a non-HTML page. When `from` has an extension other
  than `.html` and the rule isn't a copy, no stub is written and a warning is
  printed.
- A wildcard rule whose `to` is a path with exactly one `:splat` is expanded
  against the HTML pages in the route table: for `/blog/* /articles/:splat`,
  every page under `/articles/` gets a stub at `/blog/` plus the rest of its
  path. Paths already in the table are skipped, so an earlier rule or a real
  page wins. Expanded rows for drafts are skipped in `build`.
- Any other wildcard rule can't be expanded; `build` prints a warning naming
  the line, and it only works on hosts that read `_redirects`.

### Route table

`ParseAll` ends by building a `RouteTable` that maps every URL the site serves
to the one thing that produces it. Each `Route` has a kind, a URL, and the file
`build` writes it to:

| Kind | URL | File |
| --- | --- | --- |
| `page` | `/`, `/about`, `/feed.xml` | `index.html`, `about/index.html`, `feed.xml` |
| `output` | `/about.txt` | `about.txt` |
| `static` | `/img/logo.svg`, `/_assets/style.css`, `/_redirects` | the same path |
| `alias` | `/blog/2021/ngrok`, `/old.html` | `blog/2021/ngrok/index.html`, `old.html` |
| `redirect` | `/newsletter`, and each path a wildcard rule expands to | `newsletter/index.html` |

Pages with a source are page routes, written to `<path>/index.html` when the
page path has no extension (and for the root), and as-is otherwise. Bare
directories have no route. Static routes are the non-page files `walk` finds
plus the files under the root `_assets/` directory, and the root `_redirects`
file. Alias rows are added after every page and output, then a redirect row for
each `_redirects` rule without a wildcard, then the rows wildcard rules expand
to.

Lookups ignore leading and trailing slashes, so `/about` and `/about/` are the
same route. Two routes with the same URL or the same file fail `ParseAll`
with an error naming both producers, such as an output `foo.txt` and a static
file `foo.txt`, a page `foo` and a static `foo/index.html`, an alias equal to
a page's path or claimed by two pages, or an alias that matches a `_redirects`
rule. Rows expanded from wildcard rules are the exception: they never collide,
since paths already taken are skipped. Drafts stay in
the table; `Route.Draft()` reports whether the route's page is a draft, and
each mode applies its own draft rule. `Site.Routes()` returns the table from the
last `ParseAll`.

### Ordering

Every listing, from `pages()` and from `subpages`, is sorted by `sortPages`
with one total order, so it comes out the same on every build:

1. Pages without a valid `YYYY-MM-DD` `date` first, then dated pages, oldest
   first.
2. Pages with the same date, or both undated, by `slug`.
3. Pages with the same slug by `path`.

Listings that want newest first call `.reverse()` in JSX, which puts undated
pages at the end.

## Build mode

`taragen build` calls `NewSite(cwd, false).GenerateAll("_public", true)`:

1. `ParseAll()` resets the page map, parses every page path found by `walk`
   once each (a directory and its `index.*` file share one), and builds the
   route table. A route collision stops the build.
2. **Clean.** With `clobber` set, every entry in the destination is removed
   except dot-prefixed entries at its top level, so a checked-out deploy
   repository (`_public/.git`) survives. Renamed and deleted pages don't linger.
3. **Routes.** Every route is written, in order of its file: static files
   (including `_assets/` and `_redirects`) are copied verbatim, pages and
   outputs are rendered to their files, and alias and redirect rows are
   written as stubs or copies. Drafts and their outputs and aliases are
   skipped. Each page renders once, plus once for each copy made of it.
   Warnings from `ParseAll`, such as wildcard rules that can't be expanded,
   are printed first.

Any render error, including a failing partial or output template, stops the
build with a non-zero exit.

## Serve mode

`taragen serve` creates the site with drafts enabled, starts the watcher,
parses everything, and serves on port 8088 (hard-coded). `Site.ServeHTTP`:

- `/.live-reload` upgrades to a WebSocket (see below).
- Otherwise the request path is looked up in the route table. A page or output
  is rendered into a buffer on demand, including drafts, and served with its
  content type; the live reload script is injected before `</body>` only into
  `text/html` responses. A static route is served from its source file. The
  lookup and render run under the site mutex, the same one `ParseAll` takes, so
  a reload during a request can't race. A render error is a 500 with the error
  text.
- An alias of an HTML page answers with a `301` to the page's path, and an
  alias of a non-HTML page with a copy of the page. A redirect row answers
  with its rule's status, or a copy when `build` would write one.
- Paths not in the table are matched against the wildcard rules in order, on
  the cleaned, escaped request path, so a wildcard rule redirects even where
  `build` couldn't write stubs. Rules keep the request's query string when
  `to` has none. Off-site targets come only from `_redirects`, and `:splat`
  only fills in a path, so serve isn't an open redirect.
- Paths that match nothing are served from disk with `http.ServeFile`. Because
  this bypasses `walk`, underscore paths are reachable in serve mode, and a
  bare directory gets a directory listing.

### Live reload

`watchForReloads` uses `tractor.dev/toolkit-go`'s `watchfs` to recursively watch
the site directory. Any change outside dot-prefixed paths and `_public/` closes
every open reload WebSocket, so CSS and other static edits reload the browser.
A change to a page source (by the extensions in `Formats`, which includes
output templates), a directory, or an extensionless name (possibly a deleted
directory) first calls `ParseAll()` to pick up new and removed pages and
outputs; `_redirects` is extensionless, so its edits reparse too. Edits to an existing output template show on the next request, since
outputs are rendered fresh each time. The injected client script is a one-liner that
reloads the page when its socket closes:

```html
<script>(new WebSocket("/.live-reload")).onclose = () => window.location.reload();</script>
```

Closing the socket as the reload signal keeps the protocol trivial: there are
no messages, and a server restart also triggers a reload.

## Key design decisions

- **Single binary, no Node.** esbuild (as a Go library) handles the JSX syntax
  transform, and goja executes the result. Users install one binary (Homebrew
  tap or release download) and never touch npm.
- **JSX as a server-side template language.** Elements render straight to HTML
  strings through a minimal `hyper` factory instead of a React-compatible
  runtime. That keeps the runtime tiny and fast, at the cost of React
  semantics (no components with hooks, no `className`/event handling).
- **Filesystem as the configuration.** There is no config file. Routing comes
  from file paths, and behaviour is attached by underscore files whose effect is
  scoped to their directory subtree (layouts, globals, defaults, output
  templates).
- **One data map per page.** Frontmatter, `data()`, defaults, and built-in
  fields all write into the same `Data` map, which is exposed identically to Go,
  JSX (`page`), and listings (`pages()`).
- **Isolation by fresh VMs.** Every JSX render (page, layout, partial, globals
  file) runs in a new goja runtime. Pages cannot leak state into each other, but
  globals are re-evaluated for every page.
- **Parse on every render.** `Render` always re-parses, so serve mode reflects
  edits without caching logic; `ParseAll` on file changes handles the page set.
- **Custom template delimiters.** `[[ ]]` avoids conflicts with JSX braces and
  code samples in Markdown.
- **Directory = section.** A directory with or without an index is a page node,
  which makes nested listings (`pages("docs")`, `year.subpages`) fall out of the
  same model.

## Known limitations and rough edges

These reflect the current code and were verified against the example site
where noted.

- **Underscore directories.** Apart from the root `_assets/`, `walk` prunes
  every `_`-prefixed directory, which excludes `web/_blog/` from both modes;
  this appears to be how the example blog is currently disabled. Serve mode
  still serves any underscore path from disk.
- **Serialized serve.** Requests render one at a time under the site mutex,
  because rendering mutates page data. Fine for local preview.
- **Hard-coded CLI settings.** Port `8088`, output dir `_public`, and cleaning
  the output are not configurable yet (there is a TODO for the port).
- **Template partials get no helpers.** `Site.Partial` renders `.tmpl` partials
  with no function map and ignores arguments, and the template `page` helper
  assumes string values.

## Build and release

- `make build` produces `.local/bin/taragen` with the version injected via
  `-ldflags -X main.Version=v0.1-<short sha>`; `make link` symlinks it into
  `/usr/local/bin`.
- `make dist` cross-compiles for darwin, windows, and linux on amd64 and arm64
  into `.local/dist/`.
- `make web-serve` / `make web-build` run the binary against `web/`.
- `make test` runs `go test ./...`; `make design-check` runs the design check.
- CI (`.github/workflows/build.yml`) delegates to the shared
  `progrium/github-workflows` Go CLI build workflow on every push, with
  Go 1.23.1 and an extra linux-386 target.
- Tests: `go test ./...` runs the JSX rendering table tests in `jsx_test.go`,
  the page tests in `page_test.go`, the output and route table tests in
  `outputs_test.go`, the build, serve, and watcher tests in `site_test.go` and
  `serve_test.go`, and the design tooling tests.

## Design process tooling

Design changes go through proposals in `docs/design/`, as described in the
[design process README](design/README.md). The tooling lives in the repository
and uses only Go and git:

- `internal/design` parses proposal frontmatter (a strict YAML subset, with its
  own parser rather than a YAML library, so anything another parser could read
  differently is rejected), loads `docs/design/`, `SIGNERS`, and this spec, and
  implements `Check`, the computed states (ready, in progress, stalled, read
  from branch refs), and the Mermaid dependency graph. Its rules are covered by
  a table test.
- `tools/design` is the CLI: `list`, `ready`, `graph`, `new`, `log`, `sign`,
  and `check`. `sign` runs only at a terminal and asks the signer to type the
  slug.
- `design-check.yml` runs `check` on every PR.

## Dependencies

| Module | Used for |
| --- | --- |
| `github.com/evanw/esbuild` | JSX → JS transform. |
| `github.com/dop251/goja` | Embedded JavaScript runtime. |
| `github.com/yuin/goldmark`, `goldmark-highlighting` | Markdown rendering and Chroma syntax highlighting. |
| `gopkg.in/yaml.v2` | Frontmatter parsing. |
| `github.com/yosssi/gohtml` | HTML output formatting. |
| `github.com/gorilla/websocket` | Live reload socket. |
| `tractor.dev/toolkit-go` | CLI framework (`engine`, `cli`) and the `watchfs` file watcher. |
