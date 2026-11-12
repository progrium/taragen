---
slug: backlinks
title: Backlinks
status: draft
created: 2026-11-12
authors: [progrium]
depends: [page-outputs]
areas: [site, data, jsx, serve]
branch: []
---

# Proposal: Backlinks

## Abstract

Give every page the list of other pages that link to it, so layouts can show "3 logs link here" or a "linked from" list.
During `ParseAll`, taragen collects the internal links in each page's body, before layouts, and stores them as `page.links`.
It then inverts them into `page.backlinks`: the data of each page that links here, sorted like `pages()`.
Both keys are ordinary page data, so they're visible to the page, its layouts, and `pages()` results, and `serve` keeps them current through the watcher's existing `ParseAll`.

## Background

A page knows nothing about the pages that link to it.
Page data (see Page data in the [spec](../spec.md)) holds the page's own source, body, and frontmatter, and `pages()` returns other pages' data, but nothing relates one page to another except the directory tree.

The progrium.xyz redesign shows a count like "3 logs link here" under each article, and a "linked from" list on each doc.
Today a template could fake this by calling `pages("*")` and searching every other page's `body` for an `href`, but that is quadratic in the number of pages, runs again on every page render, and is fragile: it has to guess at relative links, trailing slashes, fragments, and absolute URLs on its own.

Two parts of the current design matter here:

- `ParseAll` parses pages one at a time in walk order, and a JSX body that calls `pages()` sees only the pages parsed before it. The root `index.jsx` is parsed first, so at `ParseAll` time its listing is empty. `Render` re-parses the page, so the rendered output is correct, but a link graph read from the `ParseAll` bodies would not be.
- `build` writes an extensionless page to `<path>/index.html` and serves it at `<path>/`, while `serve` answers at `<path>` without the slash. A relative `href` resolves differently against those two URLs.

The [page-outputs](page-outputs.md) proposal adds a route table, built in `ParseAll`, that maps every URL the site serves to the page, output, or static file that produces it.

## Proposal

### What counts as a link

A link is the `href` of an `<a>` element in a page's `body`: the page's own content after its format parser runs and before any layout is applied.
Links in layouts, such as nav bars and footers, don't count, since they would make every page link to every nav target.

Only pages whose `contentType` is `text/html` are link sources.
Feeds, sitemaps, and other non-HTML pages are left out, because a feed links to every post and would drown out real references.

A link counts toward a target page when its normalized URL (below) is the target page's own URL in the route table.
It doesn't count when:

- it resolves to an output of a page, such as `/foo.txt` from page-outputs, since that is the same page in another form, usually linked from the page itself;
- it resolves to a static file, or to no route at all;
- it points at the linking page itself, including `#fragment`-only links;
- the target or the linking page is a draft and the site isn't showing drafts (`build`); in `serve`, which shows drafts, they count.

When a later proposal adds routes that stand in for a page, such as alias redirects, it says whether links to them count toward that page.

A page that links to the same target several times counts once.

### Normalizing an href

Each `href` is normalized to a site path before lookup:

1. Skip it if it has a scheme other than `http` or `https` (`mailto:`, `tel:`, `javascript:`), or if it is empty.
2. If it is absolute (`https://...` or `//...`), keep it only if its origin matches the site URL in `siteURL` (below), and drop the origin. Otherwise skip it.
3. Resolve a relative `href` against the page's URL as `build` publishes it: `<path>/` for an extensionless page or the root, and `<path>` for a page whose name keeps an extension (`feed.xml`). This matches what a browser does on the deployed site.
4. Drop the query and `#fragment`, clean the path (`.` and `..` segments), and drop a trailing slash, except for the root `/`.
5. Percent-decode the path.

### `siteURL`

`siteURL` becomes a page data key with built-in meaning: the site's absolute origin, such as `https://progrium.xyz`, normally set once in the root `_defaults.jsx`.
It is read from the linking page's data, so absolute links to the site's own pages count.
When it is unset, every absolute link is treated as external.

### Building the graph

`ParseAll` gains a link pass after every page is parsed and the route table is built:

1. **Re-parse listing pages.** A page whose body called `pages()` during the first pass is parsed again, now that the whole page set exists, so listings and indexes link to everything they list. Pages that didn't call `pages()` keep their first-pass body.
2. **Collect.** For each link source, parse its `body` with `golang.org/x/net/html`'s tokenizer, normalize each `<a href>`, and set `links` in its data to the sorted, de-duplicated list of target page URLs that count, such as `["/logs/2026/ngrok", "/docs/outputs"]`.
3. **Invert.** For each page, set `backlinks` in its data to the list of pages whose `links` contain its URL, ordered by `sortPages` as `pages()` orders them.

Every page gets both keys, as empty lists when there are none.
Each `backlinks` entry is a copy of the linking page's data, the same shape as a `pages()` result, but without its own `links`, `backlinks`, or `subpages`, so the structure has no cycles:

```jsx
<p>{page.backlinks.filter(p => p.path.startsWith("/logs/")).length} logs link here</p>
<ul>{page.backlinks.map(p => <li><a href={p.path}>{p.title}</a></li>)}</ul>
```

`links` and `backlinks` replace any value a page set for them, the way `page.outputs` does.

### When the graph is read

The graph is fixed at the end of `ParseAll`.
`Render` re-parses the page as today, and keeps the `links` and `backlinks` that `ParseAll` set, rather than recomputing them from the new parse.
So a page whose body renders its own "linked from" list doesn't add links back to its referrers: those links appear only after `ParseAll`, and are never collected.

Because `body` is parsed before backlinks exist, a page body or a `pages()` call made during `ParseAll` sees empty `backlinks`; one made during `Render` sees the full graph.
Layouts always run during `Render`, so they always see it.

In `serve`, the watcher already calls `ParseAll` when a page source changes, which rebuilds the graph, so backlinks stay current on reload.

### Templates

`page "backlinks"` in a Go template returns the list, but the template `page` helper assumes strings, so for now backlinks are meant for JSX.
Template helpers for lists are out of scope.

## Rationale

**Body, not the rendered page.** Collecting from the full rendered output would count every nav and footer link, so every page in the nav would show the whole site as backlinks. The body is what the author wrote, which is what "links here" means. A link added by a layout around the content, like a "next post" link, won't count; that's the trade-off.

**`x/net/html` on the body, not goldmark's AST.** Every format ends in an HTML body, but only Markdown has an AST, and JSX and template pages would need a second path anyway. Raw HTML inside Markdown is allowed, so the AST misses `<a>` tags written by hand. The tokenizer reads all three formats one way, and `golang.org/x/net` is already in the module graph as an indirect dependency, so this adds no new module.

**Re-parse only pages that call `pages()`.** A listing parsed first sees an empty site, so its links would depend on walk order. Re-parsing every page would fix that at twice the parse cost. Only bodies that called `pages()` can change on a second pass, so taragen marks those in the `pages` builtin and re-parses just them. The `pages()` call is the only way a body can read other pages, so the mark covers every body that could change.

**A fixed graph instead of recomputing on render.** Recomputing on every render would let a page's own backlink list feed back into the graph and make the result depend on render order. Fixing it once per `ParseAll` makes it deterministic in both modes.

**Page data, not a new global.** A `backlinks(path)` global would be a second way to read page relationships. Storing the lists in page data follows "one data map per page" from the spec: the page, its layouts, and `pages()` all see them with no new API.

**Copies without nesting.** Including each referrer's own `backlinks` would form cycles (A links to B, B links to A) that goja and any JSON export would loop on. A referrer's data minus the graph keys is enough to render a title, date, and link; a template that needs more can call `pages()` on the referrer's path.

**Filtering is the template's job.** "3 logs link here" counts only logs, while a doc's "linked from" list wants everything. One full list plus `.filter()` on `path` covers both without a configuration key per section.

**Resolve through the route table.** page-outputs already maps every URL to what produces it, including the directory and `index` cases. Reusing it means one place decides what a URL is, and outputs (and later aliases) are handled by rule instead of by string matching. That is why this proposal depends on page-outputs. Resolving against `Site.pages` directly would remove the dependency, at the cost of duplicating URL rules that the route table owns.

**Build-style relative resolution.** The deployed site is what readers follow, and its extensionless pages live at `<path>/`. Resolving against `serve`'s slashless URL would count a link that is broken in production, and miss one that works there.

## Compatibility

- **Page data:** `links`, `backlinks`, and `siteURL` become keys with built-in meaning. Taragen overwrites `links` and `backlinks`; a site already using either for its own data would lose that value. `siteURL` is read, not written.
- **Build:** output files are unchanged unless a template reads the new keys. `ParseAll` does more work: one tokenizer pass per HTML page, plus a second parse of pages that call `pages()`.
- **JSX runtime:** `pages()` results gain `links` and `backlinks`. A page body evaluated during `ParseAll` now runs twice if it calls `pages()`, so side effects in such a body, like `console.log`, happen twice.
- **Serve:** backlinks are rebuilt on the existing reload trigger. HTTP behavior is unchanged.
- **CLI, templates and Markdown, distribution:** unchanged, apart from `golang.org/x/net` becoming a direct dependency.

## Security

None new. The link pass reads page bodies that taragen already produced and never fetches a URL; absolute links are compared as strings against `siteURL`.
The data it adds is other pages' data, which `pages()` already exposes to every page.

## Done criteria

- [ ] `TestLinksFromBody`: links in a page's body are collected, and links added by its layouts are not.
- [ ] `TestLinksNormalize`: relative links from `foo` and `feed.xml`, `..` segments, trailing slashes, fragments, queries, and percent-encoding all resolve to the same target URL.
- [ ] `TestLinksAbsolute`: `https://example.com/about` counts when `siteURL` is `https://example.com`, and doesn't when it is unset or different; `mailto:` and other schemes are skipped.
- [ ] `TestLinksSkip`: self links, fragment-only links, links to static files, unknown paths, and page outputs like `/about.txt` don't count.
- [ ] `TestLinksNonHTML`: a `feed.xml.jsx` linking to every post adds no backlinks.
- [ ] `TestLinksListingPage`: a root `index.jsx` that lists posts with `pages()` links to every post, regardless of walk order.
- [ ] `TestBacklinks`: `page.backlinks` holds referrers once each, sorted like `pages()`, without `links`, `backlinks`, or `subpages`, and is an empty list when nothing links in.
- [ ] `TestBacklinksDrafts`: in `build`, drafts are neither sources nor targets; in `serve` they are both.
- [ ] `TestBacklinksVisible`: a layout and a `pages()` call from another page's layout both see `backlinks`.
- [ ] `TestBacklinksStable`: a page whose body renders its own backlinks list doesn't change the graph when rendered, and rendering twice gives the same output.
- [ ] `TestServeBacklinks`: in `serve`, adding a link to page A and reloading shows A in the target's backlinks.
- [ ] The `web/` site shows a "linked from" list on docs pages.
- [ ] The spec describes `links`, `backlinks`, `siteURL`, and the link pass in `ParseAll`.

## Implementation

1. Mark pages whose body called `pages()` during parsing.
2. Add the href normalizer, with `siteURL` and build-style base URLs, and its table test.
3. Add the link pass to `ParseAll`: re-parse marked pages, collect `links` with the `x/net/html` tokenizer, and resolve them through the route table.
4. Invert into `backlinks` with the draft rule, and keep both keys across the re-parse in `Render`.
5. Update the `web/` example site and the spec.

## Open issues

- Should links from a layout ever count, for example a "related" block a layout renders from page data? This proposal says no; a page that wants such links counted can render them in its body.
- Should page-outputs be a hard dependency? Backlinks could resolve against `Site.pages` and ship first, then switch to the route table, at the cost of a second set of URL rules in the meantime.
- Is `siteURL` the right name and place for the site's origin? page-outputs' share cards will also need an absolute URL for `og:image`, so the two should agree.
