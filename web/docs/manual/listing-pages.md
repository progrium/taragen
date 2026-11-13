---
title: "Listing pages with `pages()`"
kind: manual
num: 13
order: 13
part: "content"
toc:
  - ["by-path", "By path"]
  - ["by-pattern", "By pattern"]
  - ["the-object-form-several-paths-and-filters", "The object form: several paths and filters"]
  - ["order", "Order"]
  - ["whats-in-a-listing", "What's in a listing"]
---

`pages()` returns page data objects, which is how you build indexes, archives,
feeds, and tag pages. Each result has the page's full data, including `path`,
`title`, `date`, `body`, `outputs`, `links`, `backlinks`, and a recursive
`subpages` array.

<a id="by-path"></a>

## By path

A plain path returns the children of that page:

```jsx
pages("docs")          // pages directly inside docs/
```

<a id="by-pattern"></a>

## By pattern

A path containing wildcards returns the children of every page whose path
matches:

<div class="tw"><table>
<thead><tr><th>Pattern</th><th>Returns</th></tr></thead>
<tbody>
<tr><td><code>articles/*</code></td><td class="prose">Pages two levels below <code>articles</code> (children of each <code>articles/&lt;x&gt;</code>), e.g. posts in year folders.</td></tr>
<tr><td><code>logs/202?</code></td><td class="prose">Children of <code>logs/2020</code> through <code>logs/2029</code>.</td></tr>
<tr><td><code>articles/**</code></td><td class="prose">Every page below <code>articles</code>, at any depth.</td></tr>
<tr><td><code>**</code></td><td class="prose">Every page in the site except the root.</td></tr>
</tbody>
</table></div>

`*`, `?`, and `[...]` match within one path segment and never cross a `/`. A
segment that is exactly `**` matches zero or more whole segments. (`a**` inside
a longer segment is just two `*`s.)

Patterns containing `**` leave out grouping nodes (folders with no index file),
since those only exist to hold the tree together. Plain paths, single-level
patterns, and `subpages` still include them.

Each page appears at most once in a result.

<a id="the-object-form-several-paths-and-filters"></a>

## The object form: several paths and filters

`pages()` also takes an object:

```jsx
pages({ path: ["articles/**", "logs/**", "docs/**"] })   // a mixed stream
pages({ where: { tags: "plan9" } })                         // everything tagged plan9
pages({ path: "articles/**", where: { featured: true } })
```

<div class="tw"><table>
<thead><tr><th>Key</th><th>Default</th><th>Meaning</th></tr></thead>
<tbody>
<tr><td><code>path</code></td><td class="prose"><code>&#34;**&#34;</code></td><td class="prose">A path or pattern as in the string form, or a list of them. A list returns the union, with no duplicates, sorted as one listing.</td></tr>
<tr><td><code>where</code></td><td class="prose">none</td><td class="prose">Page data keys and the values to keep. A page must match every key.</td></tr>
</tbody>
</table></div>

A `where` key matches when:

- the wanted value is a string, number, or boolean, and the page's value is
  equal to it, or is a list containing it (so `tags: "plan9"` matches
  `tags: [go, plan9]`);
- or the wanted value is a list, and any of its elements matches by the rule
  above (so `tags: ["go", "rust"]` means "tagged go or rust").

Comparison is strict: `"2026"` doesn't match `2026`, and strings are
case-sensitive. A page without the key never matches. `where` filters only the
top-level results, not their `subpages`.

`pages("x")` and `pages({ path: "x" })` return the same pages. Unknown keys, a
`path` that isn't a string or list of strings, a `where` value that is an object
or `null`, and malformed patterns throw an error that fails the render.

<a id="order"></a>

## Order

Every listing (from either form, and every `subpages` array) is sorted the same
way:

1. Pages without a valid `date` come first, then dated pages, oldest first.
2. Pages with the same date, or both undated, are ordered by `slug`.
3. Pages with the same slug are ordered by `path`.

So listings are stable from build to build. Call `.reverse()` for newest first,
which puts undated pages at the end:

```jsx
{pages("articles/**").reverse().slice(0, 10).map(a => partial("_partials/card", a))}
```

<a id="whats-in-a-listing"></a>

## What's in a listing

- In `build`, drafts are left out of every result. In `serve`, they're
  included.
- [Generated pages](/docs/manual/generated-pages) are included, and are recognizable by
  their `params` key. To leave them out: `.filter(p => !p.params)`.
- Grouping nodes are included except in `**` patterns, as described above.
