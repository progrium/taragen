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
| `page.go` | `Page`: page data model, parsing, globals/defaults loading, layout chain, rendering. |
| `format.go` | Format registry (`Formats`), `PageParser` interface, YAML frontmatter splitting. |
| `format_jsx.go` | JSX page parser and the built-in JS globals (`page`, `partial`, `pages`). |
| `format_md.go` | Markdown parser (template pass, then goldmark with class-based syntax highlighting). |
| `format_tmpl.go` | Go `text/template` parser, template helper funcs, `RenderTemplate`. |
| `jsx.go` | JSX runtime: esbuild transform, goja VM setup, `hyper` element factory, HTML serialization. |
| `reloader.go` | File watcher and WebSocket-based live reload. |
| `jsx_test.go`, `site_test.go` | Table tests for JSX rendering, and build, serve, and watcher tests on temporary sites. |
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
  layouts, partials, globals, defaults, and assets stay out of the page tree.
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
| any other `_name.jsx` / `_name.tmpl` | A partial, loaded by path via `partial("_dir/name")`. |
| `_assets/` (root only) | Static files copied verbatim to `_public/_assets/` by `build`, never rendered. |

### Page paths

Source paths are normalized into page paths by `Site.normalizePath`: strip the
site root, strip the extension, strip a trailing `/index`, and use `.` for the
root. So `about.jsx` and `about/index.jsx` both become page `about`, and
`index.jsx` becomes `.`.

Pages live in `Site.pages`, a `map[string]*Page` keyed by normalized path.
`Site.Page(path)` lazily creates an entry with `path` and `slug` seeded into its
data. Directories without an index file still get a `Page` (with `isDir: true`
and no source) so they can act as grouping nodes for `pages()` queries.

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
the future. `build` always skips drafts. `serve` constructs the site with
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
  array). A plain path returns the sorted subpages of that page. A query
  containing `*` is matched with `path.Match` against every page path and
  returns the subpages of each match, so `pages("blog/*")` returns posts nested
  under year directories.

### Markdown

Markdown first goes through the Go template parser (so frontmatter is extracted
and `[[ ... ]]` template actions run), then through goldmark with the
`goldmark-highlighting` (Chroma) extension. Raw HTML is allowed
(`html.WithUnsafe()`).

Code highlighting uses CSS classes, never inline styles, so the site's
stylesheet decides how code looks. A fenced block whose language Chroma
recognizes renders as:

```html
<pre tabindex="0" class="chroma"><code class="language-go"><span class="line"><span class="cl"><span class="kd">func</span> <span class="nf">main</span><span class="p">()</span> ...
</span></span></code></pre>
```

- `<pre>` has `class="chroma"` and `tabindex="0"`; `<code>` has
  `class="language-<lang>"`, with the info string's first word as written.
- Each line is a `span.line` wrapping a `span.cl`. Each token is a `span`
  whose class is Chroma's short name for its token type; plain text gets no
  span. No element produced by highlighting has a `style` attribute.
- Fenced blocks in a language Chroma doesn't know, fenced blocks with no
  language, indented blocks, and inline code are plain `<pre><code>` (with
  `class="language-<lang>"` when a language was given) and escaped text.
  Taragen doesn't guess languages.

The class names are Chroma's and are the public contract, the same ones Hugo
emits, so any Chroma stylesheet works unchanged (for example from
`hugo gen chromastyles`). Taragen ships and injects no highlighting CSS. Common
classes:

| Class | Token |
| --- | --- |
| `k`, `kd`, `kn`, `kc`, `kt` | Keyword, declaration, namespace, constant, type keyword |
| `nf`, `nt`, `na`, `nb` | Function, tag, attribute, and builtin names |
| `s`, `s1`, `s2`, `se` | Strings (generic, single, double quoted) and escapes |
| `m`, `mi`, `mf` | Numbers |
| `c`, `c1`, `cm` | Comments (generic, single line, multiline) |
| `o`, `p` | Operators and punctuation |
| `line`, `cl`, `hl` | Line wrapper, line content, highlighted line |

The full list is Chroma's token table:
<https://github.com/alecthomas/chroma/blob/v0.10.0/types.go>. The example site's
`web/_assets/styles/style.css` has a small palette for its brown `pre`.

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
5. Store the source in `src`, run the format's parser, merge the returned data,
   and store the result in `body`.

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

### Ordering

`sortPages` orders by `date` ascending when both pages have a parseable
`YYYY-MM-DD` date, otherwise by slug. Listings that want newest first call
`.reverse()` in JSX.

## Build mode

`taragen build` calls `NewSite(cwd, false).GenerateAll("_public", true)`:

1. `ParseAll()` resets the page map and parses every page path found by `walk`,
   once each.
2. **Clean.** With `clobber` set, every entry in the destination is removed
   except dot-prefixed entries at its top level, so a checked-out deploy
   repository (`_public/.git`) survives. Renamed and deleted pages don't linger.
3. **Assets.** Every file under the root `_assets/` directory (skipping
   dot-prefixed names) is copied verbatim to `_public/_assets/`, whatever its
   extension. `_assets` directories below the root are not copied.
4. `walk` runs again. Static files are copied to the same relative path in the
   destination. Pages are rendered, each page path once (a directory and its
   `index.*` file share one): pages without an extension (and the root) go to
   `<path>/index.html`, giving clean URLs; pages whose name keeps an extension
   (e.g. `feed.xml.jsx` → `feed.xml`) are written as-is. Pages with no source
   (bare directories) and drafts are skipped.

Any render error, including a failing partial, stops the build with a non-zero
exit.

## Serve mode

`taragen serve` creates the site with drafts enabled, starts the watcher,
parses everything, and serves on port 8088 (hard-coded). `Site.ServeHTTP`:

- `/.live-reload` upgrades to a WebSocket (see below).
- If the trimmed request path is a known page, it is rendered into a buffer,
  served with the page's content type, and the live reload script is injected
  before `</body>`. The lookup and render run under the site mutex, the same
  one `ParseAll` takes, so a reload during a request can't race. A render error
  is a 500 with the error text.
- Otherwise the request is served from disk with `http.ServeFile`. Because this
  bypasses `walk`, underscore paths such as `/_assets/...` are reachable in serve
  mode.

### Live reload

`watchForReloads` uses `tractor.dev/toolkit-go`'s `watchfs` to recursively watch
the site directory. Any change outside dot-prefixed paths and `_public/` closes
every open reload WebSocket, so CSS and other static edits reload the browser.
A change to a page source (by the extensions in `Formats`), a directory, or an
extensionless name (possibly a deleted directory) first calls `ParseAll()` to
pick up new and removed pages. The injected client script is a one-liner that
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
  scoped to their directory subtree (layouts, globals, defaults).
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
  the build, serve, and watcher tests in `site_test.go`, and the design tooling
  tests.

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
