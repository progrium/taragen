---
slug: build-and-render-fixes
title: Build and render fixes
status: implemented
created: 2026-11-12
authors: [progrium]
depends: []
areas: [cli, site, jsx, serve]
branch: [main]
---

# Proposal: Build and render fixes

## Abstract

Fix the rough edges the [spec](../spec.md) lists under known limitations that block real sites: `build` copies `_assets/` and clears stale output, JSX escapes text and attributes by default with `raw()` for trusted HTML, attributes keep their source order, broken partials fail the render, and a set of smaller fixes to serve mode, the walker, `data()` detection, and JS value handling.

## Background

The baseline spec records these as known limitations:

- `walk` prunes every `_`-prefixed directory, so `build` never copies `_assets/`, although pages link to `/_assets/...` and it works under `serve`.
- `build` passes `clobber=false`, so renamed and deleted pages keep shipping from `_public/`.
- `hyperNode.String()` writes text and attribute values verbatim, so a `&` or `"` in a title breaks `og:title` attributes and RSS XML.
- Attributes come from a Go map, so their order changes between builds and every rebuild shows spurious diffs.
- A failing `partial()` call in JSX renders its error message into the page.
- `ServeHTTP` renders without the site lock while the watcher replaces the page map, so a reload during a request races.
- A directory and its `index.*` file normalize to the same page path and are rendered twice.
- `wrapJSX` looks for the substring `data` and the first `;` anywhere in the file.
- The watcher reloads only for page sources and directories, so CSS edits need a manual refresh.
- A float child (`{1.5}`) or a numeric attribute panics.

The redesign of progrium.xyz needs all of these fixed: titles with `&`, a move of `/blog` to `/articles` that must not leave stale pages, and stable output for reviewing rebuilds.

## Proposal

### Build

- `taragen build` copies the site root's `_assets/` directory verbatim to `_public/_assets/`, skipping dot-prefixed names. Files in it are never rendered, whatever their extension. `_assets` directories below the root are not copied, and other underscore directories are still pruned.
- `taragen build` cleans `_public/` before writing: every entry is removed except dot-prefixed entries at its top level, so a checked-out deploy repository (`_public/.git`) survives.
- A page path is rendered (and parsed by `ParseAll`) once, even when both a directory and its `index.*` file map to it.

### JSX escaping

- Text children are escaped (`&`, `<`, `>`), and attribute values are escaped (`&`, `<`, `>`, `"`, `'`).
- Text inside `<script>` and `<style>` is written as is, since browsers don't decode entities there.
- A new built-in, `raw(html)`, returns a node whose HTML is inserted unescaped. `null` and `undefined` give an empty node.
- `partial()` returns its output as a raw node, and a JSX layout's `content` argument is a raw node, since both are already rendered HTML.
- Go templates and Markdown are unchanged: templates stay unescaped `text/template`, and goldmark already escapes Markdown text.

### JSX serialization

- Attributes are written in source order (the JS object's key order), as are the keys of a style object.
- Numbers are accepted as children and attribute values and are formatted the way JS prints them (`1.5`, `0.30000000000000004`).
- Booleans as children render nothing, like `null` and `undefined`, so `{cond && <b/>}` works when `cond` is `false`.
- Unsupported values (objects or functions as children, arrays as attributes) throw a JS `TypeError`, so they fail the render with an error instead of a Go panic.
- A `<newline />` marker that `replacePreNewlines` leaves inside a JS string in a `<pre>` is turned back into a newline.

### Partial errors

`partial()` in JSX throws when the partial is missing or fails to render. The error names the partial and fails the page render, so `build` exits non-zero and `serve` responds with a 500.

### Page wrapping

`wrapJSX` keeps a leading `data(...)` statement outside the page fragment only when it is the first statement in the file (after whitespace and comments). Its end is found by matching parentheses while skipping strings, template literals, and comments; a trailing `;` is optional. Pages that mention `data` elsewhere are wrapped whole.

### Serve mode

- `ServeHTTP` renders pages under the site lock, the same lock `ParseAll` takes.
- The watcher reloads the browser for any change in the site except dot-prefixed paths and `_public/`. Page sources (by the extensions in `Formats`), directories, and extensionless names also re-parse the site first.

## Rationale

- **Only the root `_assets/`.** Pages link to `/_assets/...` from the root, and copying every underscore directory would publish layouts, partials, and `_blog/`. A named directory keeps the rule simple.
- **Keep top-level dotfiles when cleaning.** Deploying from a checked-out `_public/` is common, and wiping its `.git` would be destructive. Removing the directory's contents rather than the directory also keeps a server pointed at it working.
- **Escape by default, raw by name.** Unescaped output was the source of the broken attributes and feeds, and every other JSX runtime escapes by default. Treating partials and layout content as raw keeps existing sites working, since those are the only HTML strings most sites insert. Rejected: a `dangerouslySetInnerHTML` attribute, which is more verbose and needs a wrapper element; and escaping only attributes, which leaves RSS text broken.
- **Source order over sorted order.** Both are deterministic, but source order matches what the author wrote and keeps diffs against hand-written HTML small.
- **Throw from `hyper` rather than panic.** A Go panic in a layout or partial wasn't recovered and could take down `serve`.
- **One lock for serve.** Rendering mutates page data, so a read lock isn't enough. Serializing requests costs nothing noticeable for a local preview server.

## Compatibility

- **CLI:** `build` now deletes files in `_public/` that it didn't write. `build` now fails on a broken partial, where it used to succeed with the error in the page.
- **Site conventions:** `_assets/` at the root is now part of the build output.
- **JSX runtime:** new built-in `raw`. Sites that insert HTML strings other than partials and layout content (for example `post.body` in a feed) must wrap them in `raw()`, or they are escaped. Strings that were escaped by hand (`&lt;`) are now double-escaped and should be written plainly. Attribute order in output changes once, to source order. Values that used to panic now render or throw.
- **Serve mode:** CSS and other static edits now reload the browser.
- **Templates and Markdown:** unchanged.

The example site's feed and home page are updated in this change. progrium.xyz needs `raw()` around `post.body` in `blog/feed.xml.jsx`.

## Security

Escaping by default closes injection through page data into HTML and XML output. `raw()` and partials remain unescaped by design. Cleaning `_public/` deletes files only inside that directory, and never top-level dotfiles. `_assets/` was already reachable in serve mode; it is now also published by `build`, so nothing should be kept there that isn't meant to ship.

## Done criteria

- [x] `TestBuildCopiesAssets`: root `_assets/` is copied verbatim, dotfiles, nested `_assets`, and other underscore paths are not.
- [x] `TestBuildCleansStaleOutput`: stale output is removed and `_public/.git` is kept.
- [x] `TestJSX` cases "text is escaped", "attributes are escaped", "raw is not escaped", and "script and style text is not escaped".
- [x] `TestPartialsAndContentAreNotEscaped`: partials, layout content, and `raw(page.body)` are inserted unescaped while page data is escaped.
- [x] `TestJSX` cases "attributes keep source order" and "style object keeps order", and `TestOutputIsDeterministic` across 20 builds.
- [x] `TestBuildFailsOnPartialError`: missing, throwing, and invalid partials fail the build and name the partial.
- [x] `TestServeWhileReparsing` passes under `go test -race`.
- [x] `TestBuildRendersDirIndexOnce`: a directory with an index file is rendered once.
- [x] `TestWrapJSX`: pages mentioning data, `data()` with `;` and parentheses in strings, without a semicolon, and after comments.
- [x] `TestClassifyChange`: CSS reloads without re-parsing, page sources re-parse, `_public/` and dot paths are ignored.
- [x] `TestJSX` cases "numbers", "booleans and null render nothing", "arrays of mixed children", and "newline marker in a pre string", and `TestJSXBadValuesAreErrors`.
- [x] Building `web/` with the new binary (`make build`, then `taragen build` in `web/`) produces the example site with `_assets/` and the escaped home page code sample.

## Implementation

1. Serialize JSX through a per-VM `hyper` that keeps attribute order, escapes, accepts numbers and booleans, and throws on bad values. Add `raw`.
2. Return partials and layout content as raw nodes, and throw on partial errors.
3. Rewrite `wrapJSX` to scan for a leading `data(...)` statement.
4. Copy `_assets/`, clean `_public/`, and dedupe page paths in `GenerateAll` and `ParseAll`.
5. Lock rendering in `ServeHTTP`, and widen the watcher.
6. Update the example site, the spec, and the getting started docs.
