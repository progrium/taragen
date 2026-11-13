---
title: "Generated pages"
kind: manual
num: 19
order: 19
part: "site structure"
toc:
  - ["bracketed-names", "Bracketed names"]
  - ["declaring-the-pages-with-params", "Declaring the pages with `params()`"]
  - ["what-params-can-see", "What `params()` can see"]
  - ["generated-page-data", "Generated page data"]
  - ["layouts-and-data-for-generated-pages", "Layouts and data for generated pages"]
  - ["overriding-a-generated-page", "Overriding a generated page"]
  - ["generated-pages-in-listings", "Generated pages in listings"]
  - ["drafts-and-serving", "Drafts and serving"]
---

Some pages don't have a source file of their own: one page per tag, one archive
per month. A **page template** generates many pages from one file.

<a id="bracketed-names"></a>

## Bracketed names

Write a path segment as `[name]` to make it a parameter. The source is a template
if its file name or any folder above it is a parameter:

<div class="tw"><table>
<thead><tr><th>Source</th><th>URL pattern</th><th>Example URLs</th></tr></thead>
<tbody>
<tr><td><code>tags/[tag].jsx</code></td><td class="prose"><code>/tags/[tag]</code></td><td class="prose"><code>/tags/go</code>, <code>/tags/plan9</code></td></tr>
<tr><td><code>logs/[year]/index.jsx</code></td><td class="prose"><code>/logs/[year]</code></td><td class="prose"><code>/logs/2025</code>, <code>/logs/2026</code></td></tr>
<tr><td><code>logs/[year]/[month].jsx</code></td><td class="prose"><code>/logs/[year]/[month]</code></td><td class="prose"><code>/logs/2026/11</code></td></tr>
<tr><td><code>tags/[tag]/feed.xml.jsx</code></td><td class="prose"><code>/tags/[tag]/feed.xml</code></td><td class="prose"><code>/tags/go/feed.xml</code></td></tr>
</tbody>
</table></div>

Names are lowercase letters and digits, starting with a letter, and each name
appears at most once in a path. The whole segment must be the parameter:
`[year]-archive.jsx` is an error, not a page.

Templates must be JSX. A Markdown or Go template file inside a bracketed path is
an error, and so is a static file or `_`-prefixed file inside a bracketed folder.
Put layouts, defaults, and output templates for generated pages in a real folder
above the bracketed one (see [Layouts and data](#layouts-and-data-for-generated-pages)).

<a id="declaring-the-pages-with-params"></a>

## Declaring the pages with `params()`

A template starts with a `params(...)` call that lists the parameter values to
generate. It takes an array of objects, or a function returning one, with
exactly the template's parameter names as keys:

```jsx
// tags/[tag].jsx
params(() => [...new Set(pages("**").flatMap(p => p.tags ?? []))].map(tag => ({ tag })));
data({ title: `Tagged ${page.params.tag}` });
<ul>
  {pages({ where: { tags: page.params.tag } }).reverse().map(p =>
    <li><a href={p.path}>{p.title}</a></li>
  )}
</ul>
```

```jsx
// logs/[year]/[month].jsx
params(() => pages("logs/**").map(l => ({
  year: l.date.slice(0, 4),
  month: l.date.slice(5, 7),
})));
data({ title: `${page.params.year}-${page.params.month}` });
<ul>
  {pages("logs/**")
    .filter(l => !l.params && l.date.startsWith(`${page.params.year}-${page.params.month}`))
    .map(l => <li><a href={l.path}>{l.title ?? l.date}</a></li>)}
</ul>
```

Rules:

- `params()` must be among the template's leading statements, before the
  markup, alongside any `data()` call. Anywhere else it's an error.
- Values are strings or numbers (numbers become strings). Each value must be a
  non-empty single path segment: no `/` or `\`, not `.` or `..`, and not
  starting with `.` or `_`.
- Duplicate sets are merged, so you don't need to deduplicate. An empty array
  generates no pages.
- A set with a missing or extra key, or a bad value, fails the build, naming
  the template and the set.

<a id="what-params-can-see"></a>

## What `params()` can see

`params()` runs before any generated page exists, in its own VM with your
globals and `pages()`. `page` is undefined there. `pages()` returns only real
pages, never generated ones, so one template can't feed another and the set of
pages depends only on the files on disk.

Once all pages exist, page bodies and layouts see everything: a "Tags" index
page can call `pages("tags")` and list every generated tag page.

<a id="generated-page-data"></a>

## Generated page data

A generated page's data is built like any page's, plus `params`:

<div class="tw"><table>
<thead><tr><th>Key</th><th>Value for <code>/logs/2026/11</code></th></tr></thead>
<tbody>
<tr><td><code>params</code></td><td class="prose"><code>{ year: &#34;2026&#34;, month: &#34;11&#34; }</code> (always strings)</td></tr>
<tr><td><code>path</code></td><td class="prose"><code>/logs/2026/11</code></td></tr>
<tr><td><code>slug</code></td><td class="prose"><code>11</code></td></tr>
<tr><td><code>src</code></td><td class="prose">The template&#39;s source</td></tr>
<tr><td><code>isDir</code></td><td class="prose">True if the template is an <code>index.jsx</code></td></tr>
</tbody>
</table></div>

<a id="layouts-and-data-for-generated-pages"></a>

## Layouts and data for generated pages

Globals, defaults, layouts, and output templates are found by the generated
page's path, exactly as if a real file lived there. `logs/_layout.jsx` wraps
`/logs/2026/11`, and a real `logs/2026/_defaults.jsx` applies to it. HTML
generated pages get outputs from `_outputs/` like any page, so
`/logs/2026/11.txt` exists if `_outputs/txt.jsx` does.

If a generated page's parent folder doesn't exist as a page, taragen adds a
grouping node for it, so `pages("logs/*")` reaches years that exist only through
generated pages.

<a id="overriding-a-generated-page"></a>

## Overriding a generated page

A real source at the same path wins, silently. To give one tag a hand-written
page with its own intro, add `tags/plan9.md`; every other tag keeps its generated
page. A generated page at the path of a grouping node fills it, keeping its
subpages: that's how `logs/[year]/index.jsx` gives `/logs/2026` an index.

Two templates producing the same path is an error, as is a generated page whose
URL matches a static file, output, alias, or redirect.

<a id="generated-pages-in-listings"></a>

## Generated pages in listings

Generated pages show up in `pages()` results, including `**` patterns. A home
stream built from `pages("logs/**")` will include generated month pages; skip
them with a filter on `params`:

```jsx
pages({ path: ["articles/**", "logs/**"] }).filter(p => !p.params).reverse()
```

<a id="drafts-and-serving"></a>

## Drafts and serving

`params()` follows the draft rules: in `build` its `pages()` calls never see
drafts, so a month that only holds a draft gets no page; in `serve` it does. A
generated page whose own data makes it a draft is skipped by `build`.

In `serve`, a URL that fits the pattern but wasn't generated, like
`/tags/nosuchtag`, isn't a page. Adding a tag to a Markdown file creates its page
on the next reload.
