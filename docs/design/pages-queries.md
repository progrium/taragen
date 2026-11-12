---
slug: pages-queries
title: Recursive and filtered pages() queries
status: draft
created: 2026-11-12
authors: [progrium]
depends: []
areas: [jsx, data]
branch: []
---

# Proposal: Recursive and filtered pages() queries

## Abstract

Let `pages()` list pages at any depth and filter them by page data, so a home stream or a tag hub is one call instead of several concatenated, re-sorted, and filtered listings.
A `**` path segment matches any number of segments, so `pages("articles/**")` returns every article however it's nested.
A new object form, `pages({ path, where })`, takes one pattern or a list of them and keeps only pages whose data matches `where`, such as `pages({ where: { tags: "plan9" } })` for everything tagged `plan9`.
Listings also get a total order, so pages that share a date no longer come out in a different order on every build.

## Background

`pages(query)` is the only way a page reads other pages (see `pages(query)` under JSX in the [spec](../spec.md)).
It takes a string in one of two forms:

- A plain path, such as `pages("docs")`, returns the subpages of that page: its direct children.
- A pattern containing `*`, such as `pages("blog/*")`, is matched with Go's `path.Match` against every page path, and returns the subpages of each match. Put another way, it returns the pages whose parent's path matches the pattern.

`path.Match`'s `*` never crosses a `/`, so a pattern reaches exactly one level of nesting, and a query can't say anything about page data.
The progrium.xyz redesign needs both:

- Logs live at `/logs/YYYY/MM-DD`, articles under `/articles`, and docs and projects each in their own folder. The home page is one stream of all of them, newest first.
- Tags are shared across sections, and a tag's hub page, which uses the same template as a project page, lists everything with that tag from every section.

Both work today in a template, but awkwardly: concatenate `pages("articles/*")`, `pages("logs/*")`, `pages("docs")`, and so on, filter by `tags` in JS, and sort the result again.
Each section's nesting has to be known in advance, and the one-level rule has a trap: `pages("articles/*")` returns posts under `articles/2021/` but silently drops a post at `articles/flat.md`, because a leaf page has no subpages.

Two more rough edges show up as soon as listings get bigger:

- **Unstable order.** `sortPages` compares dates and falls back to the slug only when a date is missing or unparseable. Two pages with the same date compare as equal, and they come from a Go map, so their order changes between builds. Three logs dated the same day came out in three different orders across five builds of current main. Comparing a dated page with an undated one by slug also isn't a consistent order, so a mixed listing has no defined order at all.
- **Silent errors.** A malformed pattern such as `pages("lo[gs/*")` returns an empty list instead of an error.

## Proposal

### `**` in patterns

In a pattern, a segment that is exactly `**` matches zero or more whole path segments.
Every other segment is matched as today, with `path.Match` rules, one segment at a time, so `*`, `?`, and `[...]` never cross a `/`. A `**` inside a longer segment, such as `a**`, is just two `*`s.

The result rule doesn't change: a pattern returns the pages whose parent's path matches it.
The root's path is empty for matching, so a leading `**` can match it.
So:

- `pages("articles/**")` returns every page below `articles`, at any depth, including `articles/flat` and `articles/2021/ngrok`.
- `pages("**")` returns every page in the site except the root.
- `pages("logs/*")` is unchanged: the pages exactly two levels below `logs`.

Pages without a source, the grouping nodes taragen creates for folders that have no index file, are left out of the results of any pattern that contains `**`.
They exist to hold the tree together, and `subpages` and plain-path queries still return them.
A pattern without `**` keeps returning them, as today.

Each page appears at most once in a result.

### The object form

`pages()` also accepts an object:

```js
pages({ path: "**", where: { tags: "plan9" } })
```

| Key | Default | Meaning |
| --- | --- | --- |
| `path` | `"**"` | A path or pattern, as in the string form, or a list of them. With a list, the result is the union of each one's result, with no duplicates, sorted as one listing. |
| `where` | none | An object of page data keys to wanted values. A page is kept only when every key matches. |

A key in `where` matches a page when:

- the wanted value is a string, number, or boolean, and the page's value for that key is equal to it, or is a list with an element equal to it;
- or the wanted value is a list, and any element of it matches by the rule above.

Equality is strict, as with `===`: `"2026"` doesn't match `2026`, and string comparison is case-sensitive.
A page that doesn't have the key never matches.
`where` applies only to the pages in the result, never to their `subpages`.

`pages("x")` and `pages({ path: "x" })` return the same pages.
Unknown keys, a `path` that isn't a string or a list of strings, and a `where` value that is an object or `null` are errors.

The redesign's two listings become:

```jsx
// home stream, newest first
pages({ path: ["articles/**", "logs/**", "docs/**"] }).reverse()

// tag hub, from tags/plan9.md through the shared template
pages({ where: { tags: page.slug } })
```

### Errors

A malformed pattern, in either form, throws from `pages()`, which fails the render the same way a broken partial does since build-fixes-01.

### Order

Every listing, from `pages()` in either form and from `subpages`, is sorted with one total order:

1. Pages without a valid `date` come first, then dated pages, oldest first.
2. Pages with the same date, or both without one, are ordered by `slug`.
3. Pages with the same slug are ordered by `path`.

So a listing comes out the same on every build, and `.reverse()` gives newest first with undated pages at the end.

Draft handling is unchanged: `build` leaves drafts out of every result, and `serve` includes them.

## Rationale

**Extend the pattern instead of adding a `recursive` flag.** `**` is what people already write in `.gitignore`, shells, and other site generators, and it composes with the rest of a pattern (`docs/**/api`). A flag would only say "all depths below one folder".

**Keep the parent-match rule.** Matching each page's own path would be more intuitive for new users, but it would change what every existing `*` query returns. Under the existing rule, `a/**` already means "everything below `a`", which is the common case, so the old rule costs nothing here.

**Drop grouping nodes only for `**`.** A recursive listing is a list of content, and a folder with no index has no content: in the home stream it would show up as an empty `/logs/2026` entry. Dropping them from `*` too would change existing output, so the rule is tied to the new syntax. Templates that want the tree still have `subpages`.

**A small `where` instead of a callback or a query language.** Filtering with `.filter()` in JS already works, and still will for anything `where` can't say. `where` exists for the common case, "pages with this tag" or "pages of this kind", because it reads well in a template and keeps the result sorted once. A callback argument would be `.filter()` with extra steps, and a query language (comparisons, `not`, `or` across keys) is more surface than the redesign needs. Any of these can be added later without breaking `where`.

**List membership in `where`.** Tags are a list in frontmatter, and a scalar wanted value matching a list element is what makes `{ tags: "plan9" }` work without a second operator like `contains`. The same rule works for a scalar key such as `kind: "log"`.

**`path` defaults to `**`.** A tag hub wants every section, and the default makes that the shortest call. The string form has no default because it has no optional parts.

**A list of paths instead of concatenating in JS.** Concatenating several `pages()` results leaves the template to sort and de-duplicate them, and `sortPages` isn't exposed to JS. A list in `path` keeps one sort for the whole listing.

**Fix the order here.** Unstable order existed before, but `**` and lists of paths make big mixed listings the normal case, where same-day pages and undated pages are common. Undated pages sort first so that `.reverse()`, the usual newest-first idiom, puts them at the end, and so dated pages with distinct dates keep their current order.

**No `limit` or `sort` keys.** `slice()` and `sort()` on the returned array already do this, and the redesign's listings are small.

## Compatibility

- **JSX runtime:** `pages()` accepts an object. String queries without `**` return the same pages as before. A `**` segment used to mean the same as `*`, so a site that wrote `**` expecting one level gets more pages; this is unlikely in practice. A malformed pattern now throws instead of returning an empty list.
- **Page data and ordering:** listings now have a total order. Pages sharing a date, which came out in random order, come out by slug. In a listing that mixes dated and undated pages, which had no consistent order, undated pages now come first. Listings where every page has a distinct date, or none has a date, are unchanged.
- **Templates and Markdown:** unchanged. Go templates have no `pages` helper today, and this proposal doesn't add one.
- **CLI, site conventions, serve mode, distribution:** unchanged.

## Security

None. `pages()` reads only the site's own page data, which any page can already list with `pages("*")` and filter in JS. Patterns are matched in memory against page paths and never touch the filesystem.

## Done criteria

- [ ] `TestPagesDoubleStar`: `articles/**` returns pages one and two levels down, including a leaf page directly under `articles`; `**` returns every page but the root; `docs/**/api` matches at any depth; `a**` behaves as `a*`.
- [ ] `TestPagesDoubleStarGroupingNodes`: a folder with no index is left out of `**` results, and still returned by a plain-path query and in `subpages`.
- [ ] `TestPagesSingleStarUnchanged`: `logs/*` and plain-path queries return the same pages as before this change, on the `web/` site and the test fixtures.
- [ ] `TestPagesObjectForm`: `{ path: "x" }` equals `"x"`; `path` defaults to `**`; a list of paths returns their union once each, sorted as one listing.
- [ ] `TestPagesWhere`: scalar match, list-element match, any-of with a list, AND across keys, missing keys, strict equality between `"2026"` and `2026`, and `where` not applied to `subpages`.
- [ ] `TestPagesQueryErrors`: a malformed pattern, an unknown key, a non-string `path`, and an object `where` value each fail the render with an error naming the query.
- [ ] `TestPagesOrder`: same-date pages order by slug, then path; undated pages come before dated ones; the same listing is identical across 20 builds.
- [ ] `TestPagesDrafts`: drafts are left out of every query form in `build` and included in `serve`.
- [ ] The `web/` site lists recent posts with `**` and has a tag page that uses `where`.
- [ ] The spec describes `**`, the object form, query errors, and the listing order.

## Implementation

1. Replace `sortPages` with the total order, and add `TestPagesOrder`.
2. Add a segment matcher with `**` support on top of `path.Match`, returning errors for malformed patterns, and use it in `Site.Pages`, with the grouping node rule.
3. Accept the object form in the `pages` builtin: validate it, union and de-duplicate a list of paths, apply `where`, sort once, and throw on errors.
4. Update the `web/` site and the spec.

## Open issues

- Generated pages, proposed separately, would give folders like `/logs/2026` a source, so they'd appear in `logs/**`. A stream would then filter them out by path or with a `where` key such as `kind`. Should generated pages instead be left out of `**` like grouping nodes, or is that the generated pages proposal's call?
- Each result still carries its recursive `subpages`, so `pages("**")` copies deep trees for every page near the root. That's fine at the redesign's size; a `subpages: false` key could be added if it isn't.
