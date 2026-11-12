---
slug: aliases
title: Aliases and redirects
status: implemented
created: 2026-11-12
authors: [progrium]
depends: [page-outputs]
areas: [site, data, serve]
branch: [main]
---

# Proposal: Aliases and redirects

Nov 12, 2026 · Jeff Lindsay, h.d.a.

## Abstract

Let a page list the old URLs it used to live at, as `aliases` in its data, and let a site list other redirects in a root `_redirects` file in Netlify's format.
`build` writes a small redirect stub page at each old URL, so redirects work on hosts that can't send real ones, like GitHub Pages, and copies `_redirects` through for hosts that can.
`serve` answers the same URLs with real redirects.
Both add rows to the route table from [page-outputs](page-outputs.md), so collisions with pages, outputs, and static files are build errors.

## Background

Taragen has no way to redirect a URL (see Build mode and Serve mode in the [spec](../spec.md)).
Every file in `_public/` comes from a page source or a static file, and `build` now cleans `_public/` first, so a page that moves leaves nothing behind at its old URL.

The progrium.xyz redesign moves the blog from `/blog/<year>/<slug>` to `/articles/<year>/<slug>`, and the site is deployed to GitHub Pages, which ignores host redirect files.
Today the only fix is a hand-written stub page per old URL, about twenty files that drift from the pages they point to.

Other static site generators handle this two ways: per-page aliases (Hugo's `aliases`, Jekyll's `redirect_from`), and a host redirects file (Netlify's and Cloudflare Pages' `_redirects`).
Per-page aliases keep a redirect next to its content, so it moves when the content moves.
A redirects file covers what has no page behind it: an off-site URL, or a pattern over many pages.

The [page-outputs](page-outputs.md) proposal adds a route table that maps every URL the site serves to what produces it, shared by `build` and `serve`.
It leaves aliases out of scope because their URLs come from page data rather than a file name suffix.

## Proposal

### Page aliases

`aliases` becomes a page data key with built-in meaning: a list of URL paths that redirect to the page.

```yaml
title: Building your own Ngrok in 130 lines
aliases: [/blog/2021/building-your-own-ngrok-in-130-lines]
```

A single string is read as a list of one.
Any other type, or a list entry that isn't a string, is an error naming the page.

Aliases are read once the page is parsed, so they can come from `_defaults.jsx`, frontmatter, or `data()` in the page.
Layouts run only at render time, so aliases set by a layout are ignored.

Because aliases are ordinary page data, one line in a `_defaults.jsx` covers a whole directory.
For the redesign, `articles/_defaults.jsx` keeps every old blog URL working:

```js
aliases = [page.path.replace(/^\/articles/, "/blog")]
```

That gives `/articles/2021/ngrok` the alias `/blog/2021/ngrok`, and `/articles` (from `articles/index.jsx`) the alias `/blog`.
A page that sets its own `aliases` replaces the default rather than adding to it, as with any other key.

Only pages that are built get alias routes: pages with a source, not bare directories.
A draft's aliases are drafts too: `build` skips them and `serve` serves them.

### Alias paths

Each alias is normalized before use:

- It must start with `/`.
- A trailing `/` is removed, except for `/` itself.
- It must not contain `?`, `#`, `\`, an empty segment (`//`), or a `.` or `..` segment.

An invalid alias is an error naming the page and the alias.
Duplicate entries in one page's list are dropped.
After normalization, taragen replaces `page.aliases` with the normalized list, so layouts and `pages()` results see clean values, for example to list a page's old URLs or build a sitemap.

An alias maps to an output file the way a page path does:
an alias whose last segment has no extension is written to `<alias>/index.html`, and one with an extension, such as `/old.html`, is written as is.

### Redirect stubs

For an alias of an HTML page (`contentType` of `text/html`), `build` writes a stub page:

```html
<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>Redirecting to /articles/2021/ngrok</title>
<link rel="canonical" href="/articles/2021/ngrok">
<meta name="robots" content="noindex">
<meta http-equiv="refresh" content="0; url=/articles/2021/ngrok">
</head>
<body><a href="/articles/2021/ngrok">/articles/2021/ngrok</a></body>
</html>
```

The target is the page's `path`.
It is escaped as an attribute value and as text, with the same escaping rules as JSX output.
Stubs don't run layouts, globals, or output templates, and they are the same on every build.

`serve` answers a request for an alias of an HTML page with a `301 Moved Permanently` to the page's path.

### Aliases of other pages

A stub only redirects a browser, so it would break a feed reader or a script that fetches the old URL.
For an alias of a page whose content type isn't `text/html`, such as `feed.xml.jsx`, both `build` and `serve` produce a copy of the page's output at the alias, with the page's content type.
That keeps an old feed URL working for existing subscribers.

### The redirects file

A site may have a `_redirects` file at its root, in a subset of Netlify's format.
Each line is a rule, blank lines are ignored, and `#` starts a comment line:

```
# from            to                               status
/newsletter       https://buttondown.com/progrium  302
/blog/*           /articles/:splat
/talks            /articles?tag=talks
```

A rule is `from to [status]`, separated by whitespace:

- `from` is a path, normalized like an alias. It may end in `/*`, a wildcard matching every path below it; `*` may appear nowhere else.
- `to` is a path starting with `/` or an absolute `http` or `https` URL. It may contain `:splat`, which is replaced by what the wildcard matched, only when `from` ends in `/*`. A path target may carry a query string, which is passed through.
- `status` is `301`, `302`, `307`, or `308`, and defaults to `301`.

Anything else in Netlify's format is an error naming the file and line: other statuses (such as `200` rewrites and `404`), the `!` force suffix, named placeholders like `:year`, query parameter conditions, and extra fields.

Only the root `_redirects` file is read.
`build` copies it verbatim to `_public/_redirects`, so hosts that read it send real redirects, and also writes stubs from it so hosts that don't still work.

### Stubs from the redirects file

`build` writes stubs from the rules, like alias stubs:

- **A rule without a wildcard** writes one stub at `from` that points to `to`. If `to` is a path that the route table maps to a non-HTML page, the rule writes a copy of that page instead, as for aliases.
- **A wildcard rule whose `to` is a path containing `:splat`** is expanded against the HTML pages in the route table. For `/blog/* /articles/:splat`, every built page whose path starts with `/articles/` gets a stub at `/blog/` plus the rest of its path. Drafts are not expanded in `build`.
- **Any other wildcard rule**, such as one pointing off-site or without `:splat`, has nothing to expand against. `build` writes no stubs for it and prints a warning naming the line, since it only works on a host that reads `_redirects`.

`serve` answers every rule directly, with its status, and keeps the query string of the request when `to` has none.

A rule whose `from` has an extension other than `.html`, and whose `to` isn't an internal non-HTML page, gets no stub and a warning, since an HTML stub there would be served with the wrong type.

### Route table

Aliases and redirect rules add rows to the route table from [page-outputs](page-outputs.md):

- an **alias** row for each alias of each built page, pointing at the page;
- a **redirect** row for each rule without a wildcard, and for each path a wildcard rule expands to.

Rows from aliases and from rules without a wildcard follow the existing collision rule: a URL claimed by two producers fails the build, naming both.
That includes an alias equal to a page's own path, an alias that two pages claim, and an alias that matches a rule in `_redirects`.

Rows expanded from a wildcard rule are the exception: a path that is already in the table is skipped, and the first matching rule wins between rules.
This matches Netlify, which applies a rule only when no file exists at its path, and lets a site keep a real page at `/blog/2021/something` under a `/blog/*` rule.

In `serve`, a request is looked up in the route table first, then matched against the wildcard rules in order, then served from disk as before.
So a wildcard rule redirects in `serve` even when `build` couldn't write stubs for it.

### Watching

`_redirects` has no format extension, so the watcher also runs `ParseAll` when it changes, the same as for a page source.

## Rationale

**Aliases on the page, not only a redirects file.** A redirect that lives next to its content moves and is deleted with it, and the filesystem is taragen's configuration. The redirects file stays for what has no page behind it.

**A redirects file at all.** Off-site redirects (`/newsletter` to a newsletter host) and old URLs with no current page have nowhere else to go. Without it, a site would add a page per off-site redirect just to hang an alias on.

**Netlify's format and the `_redirects` name.** It's the format the hosts that read redirect files already use, so one file serves both the host and the stubs. The leading `_` keeps it out of the page tree with no change to the walker, the same way `_assets/` is handled at the root. Another name, such as `redirects.txt`, would be published as a static file and still need copying to `_redirects` for Netlify.

**A strict subset.** Rewrites (`200`), custom `404`s, and conditions can't be expressed as static stubs, and silently passing them through would make `serve` and GitHub Pages disagree with Netlify. Rejecting them keeps the three in step. Allowing host-only lines is an open issue.

**Wildcards expand against targets.** The old URLs under `/blog/*` no longer exist as files, so the only way to know which stubs to write is from the pages the rule points to. That covers the common "this section moved" case, and `serve` still matches any path.

**Stubs, not host-specific config.** GitHub Pages, S3 websites, and plain file servers can't send redirects, so a stub is the only thing that works everywhere. Meta refresh with a canonical link is what Hugo and Jekyll's redirect plugin write, and search engines treat it as a permanent redirect.

**A copy for non-HTML aliases.** A feed reader doesn't follow a meta refresh. A copy keeps the old URL working at the cost of serving the same feed twice. Making it an error was the other option, but moving a feed is exactly the case the redesign has.

**Not an output template.** The [page-outputs](page-outputs.md) discussion called aliases a built-in output. Outputs have one URL per page from a file name suffix and appear in `page.outputs`; aliases have any number of URLs from page data. They share the route table, not the output mechanism, and opting out of outputs doesn't remove aliases. A page drops its aliases with `aliases: []`.

**Stubs aren't customizable.** A styled "this page has moved" page is rarely seen, since the refresh is immediate. A fixed stub needs no template lookup and can't fail to render.

**No JavaScript in stubs.** A script could carry the URL fragment (`#section`) across the redirect, which meta refresh drops. It's left as an open issue so stubs stay inert.

## Compatibility

- **Page data:** `aliases` becomes a key with built-in meaning, and taragen rewrites it to the normalized list. A site already using `aliases` for its own data would get redirects and errors on invalid paths.
- **Site conventions:** `_redirects` at the root becomes a special file, read and copied by `build`. A site with a `_redirects` file today has it ignored, so it would start taking effect.
- **Build:** sites with neither feature build the same files. New files are written at alias and redirect paths, and new collisions fail the build. Wildcard rules that can't be expanded print warnings.
- **Serve:** alias and redirect paths answer with redirects or copies instead of 404s.
- **JSX runtime, templates and Markdown, CLI, and distribution:** unchanged.

## Security

Alias and redirect paths come from site files and are normalized to reject `..`, `.`, empty segments, and backslashes, so no stub can be written outside `_public/`.
Collisions are errors, so a stub can't overwrite a page, output, or static file.
Redirect targets are escaped in stubs, so a target containing `"` or `<` can't inject markup.
Off-site targets are allowed only in `_redirects`, which the site author writes, so `serve` doesn't become an open redirect for request-controlled URLs; `:splat` substitutes only into the target's path.

## Done criteria

- [x] `TestAliasStub`: a page with `aliases: [/old/path]` builds `_public/old/path/index.html` with the refresh, canonical link, fallback link, and `noindex`, pointing at the page's path.
- [~] `TestAliasDefaults`: an `articles/_defaults.jsx` deriving aliases from `page.path` gives every article and the `articles` index a stub under `/blog`, and bare directories get none. Waived for the index only: `articles/index.jsx` is page `/articles`, whose directory is the root, so `articles/_defaults.jsx` doesn't apply to it (see the 2026-11-12 update). Every article gets its stub and bare directories get none.
- [x] `TestAliasNormalize`: trailing slashes are removed, a string becomes a list, duplicates are dropped, `page.aliases` holds the normalized list, and `..`, `//`, `?`, and relative paths fail the build.
- [x] `TestAliasEscaping`: a page path with `&` and `"` is escaped in the stub.
- [x] `TestAliasExtension`: `/old.html` is written as is; an alias of `feed.xml.jsx` gets a copy with the feed's content.
- [x] `TestAliasDrafts`: a draft's aliases are skipped in `build` and redirect in `serve`.
- [x] `TestAliasCollision`: an alias matching another page, another page's alias, a static file, an output, or a `_redirects` rule fails the build naming both.
- [x] `TestRedirectsFile`: `_redirects` is copied verbatim, a rule without a wildcard writes a stub (including to an off-site URL), and unsupported lines fail with the line number.
- [x] `TestRedirectsWildcard`: `/blog/* /articles/:splat` writes a stub for every built page under `/articles/`, skips paths already in the route table, and an off-site wildcard rule warns and writes nothing.
- [x] `TestServeRedirects`: `serve` sends a 301 for an alias, each rule's status for `_redirects` rules including wildcards, a copy for a feed alias, and picks up `_redirects` edits after a reload.
- [x] The progrium.xyz redesign's `articles/_defaults.jsx` one-liner keeps every old `/blog/<year>/<slug>` URL working after `taragen build`, checked by building the site and listing the stubs.
- [x] The spec describes `aliases`, `_redirects`, stubs, and their route table rows.

## Implementation

1. Read and normalize `aliases` after each page is parsed, rewrite `page.aliases`, and add alias rows to the route table with collision errors.
2. Write HTML stubs and non-HTML copies for alias rows in `build`; answer them in `serve`.
3. Parse `_redirects`, validate the supported subset, and copy it in `build`.
4. Add redirect rows for rules without a wildcard, then expand wildcard rules against the route table, with warnings for rules that can't be expanded.
5. Match wildcard rules in `serve` after the route table, and rerun `ParseAll` when `_redirects` changes.
6. Add an alias and a `_redirects` file to the `web/` example site, and update the spec.

## Open issues

- Should stubs carry the URL fragment across with a one-line script, at the cost of stubs that aren't inert?
- Should `_redirects` allow host-only lines (rewrites, `!`, conditions) behind an explicit marker, copied through but ignored by stubs and `serve`?
- Should stubs use an absolute canonical URL? Taragen doesn't know the site's base URL today, so they use the path.

## Updates

- 2026-11-12: Implemented. `_defaults.jsx` applies to pages from the root down to the page's directory, and an index page's directory is its parent, so the `articles/_defaults.jsx` one-liner covers every article but not `/articles` itself; the redesign adds `aliases: /blog` to `articles/index.jsx` for that. A trial build of progrium.xyz with `blog/` moved to `articles/` wrote stubs for all 15 published `/blog/<year>/<slug>` posts and `/blog/archive`, a copy of the feed at `/blog/feed.xml`, and skipped the drafts.
- 2026-11-12: Wildcard expansion uses the HTML page rows only, so `/blog/* /articles/:splat` doesn't copy `/articles/feed.xml` to `/blog/feed.xml`; an alias or a rule without a wildcard does that. Build prints `copied:` for alias and redirect copies and `redirect:` for stubs.
