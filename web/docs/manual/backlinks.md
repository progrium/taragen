---
title: "Backlinks"
kind: manual
num: 20
order: 20
part: "site structure"
toc:
  - ["links-and-backlinks", "`links` and `backlinks`"]
  - ["what-counts-as-a-link", "What counts as a link"]
  - ["how-hrefs-are-resolved", "How hrefs are resolved"]
  - ["when-backlinks-are-available", "When backlinks are available"]
---

Taragen records which pages link to which, so a page can show what links to it:

```jsx
// in articles/_layout.jsx
{page.backlinks.length > 0 && <aside>
  <h2>Linked from</h2>
  <ul>{page.backlinks.map(p => <li><a href={p.path}>{p.title}</a></li>)}</ul>
</aside>}
```

Filter by path for a section-specific count:

```jsx
<p>{page.backlinks.filter(p => p.path.startsWith("/logs/")).length} logs link here</p>
```

<a id="links-and-backlinks"></a>

## `links` and `backlinks`

Every page gets two lists, empty when there's nothing:

- `page.links`: the sorted URLs of site pages this page's body links to, such as
  `["/docs/outputs", "/logs/2026/ngrok"]`.
- `page.backlinks`: the pages that link to this page, in the usual
  [listing order](/docs/manual/listing-pages#order). Each entry is a copy of the linking page's data, like
  a `pages()` result, but without `links`, `backlinks`, or `subpages` (so there
  are no cycles). To get more about a referrer, call `pages()` on its path.

`pages()` results carry both lists too.

<a id="what-counts-as-a-link"></a>

## What counts as a link

A link is the `href` of an `<a>` in a page's **body**: its own content, before
layouts. Nav bars, footers, and other links added by layouts don't count, or
every page would link to every nav target. If you want a link counted, put it in
the page body.

Only HTML pages are link sources; feeds and sitemaps link to everything and
would drown out real references.

A link counts toward a page when it resolves to that page's own URL. It doesn't
count when it:

- points at an output, such as `/articles/x.txt`, or a static file, or nothing;
- points at an alias or redirect URL (link to the page's current path instead);
- points at the linking page itself, including `#section` links;
- involves a draft, in `build`. In `serve`, drafts count.

Several links from one page to the same target count once.

<a id="how-hrefs-are-resolved"></a>

## How hrefs are resolved

- `mailto:`, `tel:`, `javascript:`, and other non-HTTP schemes are skipped.
- Absolute links (`https://...`, `//...`) count only if their origin matches
  `siteURL`. Set `siteURL` in your root `_defaults.jsx` so full links to your own
  pages count:

  ```js
  siteURL = "https://example.com";
  ```

  Without it, every absolute link is treated as external.
- Relative links are resolved the way a browser resolves them on the built
  site, where a clean-URL page lives at `<path>/`. So in `/docs/intro`,
  `href="setup"` means `/docs/intro/setup`, and you'd write `href="../setup"`
  or `href="/docs/setup"` to reach a sibling.
- Query strings and `#fragments` are dropped, `.` and `..` are resolved, a
  trailing slash is dropped, and the path is percent-decoded.

<a id="when-backlinks-are-available"></a>

## When backlinks are available

The link graph is computed once per full parse of the site: in `build`, and in
`serve` after each change. Index pages that call `pages()` are parsed a second
time, once all pages exist, so their links to everything they list count.

Layouts always see the full graph, as does any page body in the final output.
Links rendered from `backlinks` themselves (a "linked from" list) are not
collected, so a backlinks section never feeds back into the graph.

One side effect: a page body that calls `pages()` runs twice per parse, so
something like `console.log` in it prints twice.
