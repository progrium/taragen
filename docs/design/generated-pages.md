---
slug: generated-pages
title: Generated pages from bracketed templates
status: draft
created: 2026-11-12
authors: [progrium]
depends: [page-outputs]
areas: [site, data, jsx, serve]
branch: []
---

# Proposal: Generated pages from bracketed templates

## Abstract

Let one JSX file render many pages that have no source file of their own, such as month archives and tag listings.
A file or directory named with a bracketed parameter, like `logs/[year]/[month].jsx`, is a template.
Its leading `params(...)` call returns the parameter values, and taragen renders the template once per set, at `/logs/2026/11`, `/logs/2026/10`, and so on, with the values in `page.params`.
`params` runs after the real pages are parsed and sees only them, a real source always wins over a generated page at the same path, and two templates producing one path is a build error.
Generated pages are rows in the route table from [page-outputs](page-outputs.md), so `build` writes them, `serve` renders them on demand, and they get outputs like any other page.

## Background

Every page in taragen comes from one source file, or is a bare directory with no source (see Site and file conventions and Page paths in the [spec](../spec.md)).
The page set is whatever `walk` finds on disk, and `build` skips bare directories, so a directory like `logs/2026/` that holds only log entries has no page at `/logs/2026`.

The progrium.xyz redesign needs pages that follow from the content rather than from files:

- **Month pages** at `/logs/2026/11`, listing that month's logs. Log files are named by day, like `logs/2026/11-09.md`, so no directory exists per month.
- **Year pages** at `/logs/2026`, filling the bare `logs/2026/` directory.
- **Tag pages** at `/tags/<tag>` for every tag in use. A few tags deserve a hand-written hub page, like `tags/plan9.md`, and the rest only need a plain listing.

Writing these by hand means a new file every month and for every new tag, and they drift from the content.
The [page-outputs](page-outputs.md) proposal names generated pages as a follow-up that adds rows to its route table, and this is that proposal.

Two parts of the current design shape it.
First, `ParseAll` parses every page and then the pages render, so `pages()` during a render sees the whole site, but nothing runs between the two steps.
Second, `pages()` hides drafts when the site is not in drafts mode, so anything computed from `pages()` in `build` already ignores drafts.

## Proposal

### Templates

A path segment written as `[name]`, the whole segment and nothing else, is a parameter.
`name` is a lowercase word of letters and digits, starting with a letter, and a name appears at most once in a path.

A page source is a template when its file name, before the extensions, is a parameter, or when any directory above it within the site is a parameter:

| Source | URL pattern | Parameters |
| --- | --- | --- |
| `tags/[tag].jsx` | `/tags/[tag]` | `tag` |
| `logs/[year]/index.jsx` | `/logs/[year]` | `year` |
| `logs/[year]/[month].jsx` | `/logs/[year]/[month]` | `year`, `month` |
| `tags/[tag]/feed.xml.jsx` | `/tags/[tag]/feed.xml` | `tag` |

The URL pattern is the source's page path, normalized as for any page (the format extension and a trailing `/index` stripped, and an extra extension like `.xml` kept).
A template is never a page itself, and `walk` doesn't register its bracketed path.

Templates must be `.jsx`.
A `.md` or `.tmpl` source whose path contains a parameter is a build error, as is any static file or `_`-prefixed file inside a bracketed directory.
A segment that has brackets but isn't exactly `[name]`, like `[year]-archive.jsx`, is a build error too, so a typo can't silently become a page.

### `params()`

A template's leading statements, after whitespace and comments, must include one `params(...)` call.
Its argument is an array of parameter sets, or a function with no arguments that returns one.
Each set is an object with exactly the template's parameter names as keys:

```jsx
// logs/[year]/[month].jsx
params(() => pages("logs/*").map(l => ({
  year: l.date.slice(0, 4),
  month: l.date.slice(5, 7),
})));
data({ title: `${page.params.year}-${page.params.month}` });
<ul>
  {pages("logs/*")
    .filter(l => l.date.startsWith(`${page.params.year}-${page.params.month}`))
    .map(l => <li><a href={l.path}>{l.title ?? l.date}</a></li>)}
</ul>
```

`JSXParser` keeps every leading `params(...)` or `data(...)` call outside the page fragment, in source order, using the matching that already finds the leading `data()` call.

Values are strings or numbers, and numbers are converted to strings as JS prints them.
Each value must be a non-empty path segment: no `/` or `\`, not `.` or `..`, and not starting with `.` or `_`.
A set with a missing or extra key, or a value that breaks these rules, fails the build with the template and the set named.
Identical sets are merged, so `params` doesn't need to deduplicate, and an empty array generates no pages.

`params()` anywhere but a template's leading statements is an error.
When a template renders, `params()` does nothing.

### Parsing in two passes

`ParseAll` gains a second pass:

1. **Real pages.** `walk` finds and parses every real page and bare directory, as today, skipping templates.
2. **Parameters.** For each template, taragen runs only its `params(...)` statement, in a fresh VM with the globals from the template's real ancestor directories and the built-in `pages()`, and `page` undefined. `pages()` sees the real pages from step 1 and none of the generated ones, so one template's output can't feed another's.
3. **Register.** Each parameter set fills the URL pattern into a page path. Collisions are resolved (below), and the surviving paths are registered in `Site.pages`, seeded with `path`, `slug`, and `params`.
4. **Parse generated pages.** Each generated page is parsed from its template's source.

Templates are processed in sorted source path order, so the result doesn't depend on directory listing order.
Because rendering happens after `ParseAll`, layouts and page bodies that call `pages()` while rendering see generated pages too, which is how a "Tags" index can list every tag page.

### Generated page data

A generated page's data is built like a real page's, with one more seeded key:

| Key | Meaning |
| --- | --- |
| `params` | The page's parameter set, as strings, such as `{ year: "2026", month: "11" }`. Only generated pages have it. |
| `slug` | The last segment of the page path, as for any page: the last parameter value for `[month].jsx`, the year for `[year]/index.jsx`. |
| `src` | The template's source. |
| `isDir` | True when the template is an `index.jsx`, as for real index files. |

Globals, defaults, layouts, and output templates are found by the generated page path, exactly as for a real source at that path.
So `logs/_layout.jsx` wraps `/logs/2026/11`, and a real `logs/2026/_defaults.jsx` applies to it.

When a generated page's parent directory has no page, taragen adds a bare directory page for it, as `walk` does for real directories, so `pages("logs/*")` reaches a year that exists only through generated pages.

### Collisions

Generated pages join the route table from [page-outputs](page-outputs.md), and its rule that each URL has one producer holds, with one exception:

- **A real source wins.** If a real page source already has the path, the generated page is dropped without an error. This is how `tags/plan9.md` replaces the generated `/tags/plan9` while every other tag keeps its listing.
- **A bare directory is filled.** A bare directory has no source, so a generated page at its path takes its place and keeps its subpages. This is how `logs/[year]/index.jsx` fills `/logs/2026`.
- **Two templates is an error.** Two templates producing the same path fail the build, naming both templates and the path.
- **Everything else is an error.** A generated page whose URL matches a static file or another page's output fails the build, naming both, as for any route table collision.

### Drafts

`params` runs `pages()` under the site's draft mode, so in `build` it never sees drafts, and a month that holds only a draft gets no page.
In `serve`, which shows drafts, the same month does get a page.

A generated page is itself a draft if its data says so, through `draft: true` or a future `date`, and `build` skips it like any draft.

### Build and serve

`build` writes generated pages from the route table like real pages: an extensionless path goes to `<path>/index.html` and a kept extension is written as-is.
Generated pages that are `text/html` get outputs from `_outputs/`, so `/logs/2026/11.txt` exists when `_outputs/txt.jsx` does.

`serve` renders a generated page on request from the route table, with the live reload script injected as for any HTML page.
A path that fills a pattern but isn't in the table, like `/tags/nosuchtag`, isn't a page and falls through to disk as today.
Every change that triggers `ParseAll` reruns `params`, so a new tag in a Markdown file gets its page on the next reload.

### Out of scope

- **Catch-all parameters** like `[...rest]`, which would span several segments.
- **Pagination** helpers. A template like `logs/page/[n].jsx` works with this proposal, but taragen adds nothing to compute pages of results.
- **Generated pages from Markdown or Go templates.** Frontmatter can't hold a function, so these need a different way to declare parameters.

## Rationale

**Brackets in file names, not a generator script.** The alternative is one `_generate.jsx` per directory, or per site, that returns any list of pages with their paths and contents. It is more flexible, but the file tree no longer shows which URLs exist, and paths become arbitrary data that has to be validated after the fact. With brackets, the folder layout reads as the site map, each template generates one shape of URL, and taragen's "the filesystem is the configuration" decision holds. Bracketed routes are also a familiar convention from other file-routed frameworks. This was the choice made when the design was discussed.

**A `params()` call, not a `params` key in `data()`.** `data()` runs when the page parses, and a generated page's `data()` usually reads `page.params`, so parameters have to exist before `data()` runs. A separate statement can run alone in the parameters pass, without the page, which keeps the two passes distinct. It reuses the leading-statement rule `data()` already has.

**`params` sees only real pages.** If generated pages could see each other, the result would depend on the order templates run, and a template could feed itself. Keeping the parameters pass to real pages makes the page set a function of the files on disk. Listings that need generated pages, like a tag index, read them at render time, which comes later.

**Real sources win silently.** Overriding one generated page with a hand-written one is the point of the tag hub use case, so it can't be an error, and a warning would fire on every build of a site that uses it on purpose. Two templates claiming one path has no intended use, so it's an error.

**Generated pages stay in `pages()`.** Hiding them would make the tag index impossible without a second query function. They're identifiable by their `params` key, so a listing that wants only real pages filters on it.

**Strict values.** Parameter values become directory names in `_public/`. Rejecting `/`, `..`, and leading `.` or `_` keeps a generated page inside its pattern's place in the tree and out of paths the walker treats specially.

## Compatibility

- **Site conventions:** a path segment written as `[name]` becomes a parameter. A site with bracketed file or directory names would see them become templates, or build errors. No known site has them.
- **Page data:** `params` becomes a key with built-in meaning, seeded on generated pages. A real page that sets its own `params` keeps it, but listings filtering on `params` would mistake it for a generated page.
- **JSX runtime:** `params()` is a new built-in global, valid only as a template's leading statement. A site global named `params` is replaced, as built-ins win over site globals.
- **Build:** sites without bracketed names build the same files. Generated pages add files to `_public/`.
- **Serve:** generated URLs are served. Other requests behave as before.
- **CLI and distribution:** unchanged.

## Security

Templates run in the same sandboxed goja VM as every other JSX file, and `params` gets the same `pages()` global pages already have, so nothing new becomes reachable to site code.
Generated paths are built from site code, so their values are validated as single path segments that can't start with `.` or `_`, and the route table rejects collisions, so a generated page can't be written outside `_public/` or over another file.

## Done criteria

- [ ] `TestGeneratedTag`: `tags/[tag].jsx` with params from page tags builds one `tags/<tag>/index.html` per tag, each seeing its own `page.params.tag`.
- [ ] `TestGeneratedNested`: `logs/[year]/[month].jsx` builds `/logs/2026/11` and `/logs/2026/10` from dated logs, with `slug` set to the month.
- [ ] `TestGeneratedFillsDir`: `logs/[year]/index.jsx` fills the bare `logs/2026/` directory, and `pages("logs/2026")` from it still lists the real logs.
- [ ] `TestGeneratedMissingParent`: a generated `/logs/2027/01` with no `logs/2027/` on disk gets a bare directory page that `pages("logs/*")` reaches.
- [ ] `TestGeneratedRealWins`: a real `tags/plan9.md` replaces the generated `/tags/plan9` without an error.
- [ ] `TestGeneratedCollision`: two templates producing one path fail `build` naming both; a generated path matching a static file fails naming both.
- [ ] `TestGeneratedParamsRealOnly`: `pages()` inside `params` returns no generated pages, and `pages()` in a page body at render time does.
- [ ] `TestGeneratedDrafts`: a month holding only a draft log gets no page in `build` and gets one in `serve`; a template setting `draft: true` is skipped by `build`.
- [ ] `TestGeneratedParamErrors`: a missing key, an extra key, a value with `/`, a value of `..`, and a value starting with `_` each fail the build naming the template.
- [ ] `TestGeneratedBadNames`: `[year]-archive.jsx`, `tags/[tag].md`, and a static file inside `logs/[year]/` each fail the build.
- [ ] `TestGeneratedDedupe`: repeated identical sets produce one page, and an empty array produces none.
- [ ] `TestGeneratedOutputs`: with `_outputs/txt.jsx`, a generated page gets `page.outputs.txt` and its `.txt` file is built.
- [ ] `TestGeneratedLayouts`: a generated page is wrapped by the layouts and gets the defaults found along its generated path.
- [ ] `TestServeGenerated`: `serve` renders `/tags/<tag>` on request, returns the disk fallthrough for an unknown tag, and picks up a new tag added to a Markdown file after a reload.
- [ ] The `web/` site adds a generated page, such as tag pages for the docs, and `make web-build` writes it.
- [ ] The spec describes templates, `params()`, `page.params`, the two-pass `ParseAll`, and the collision rules.

## Implementation

1. Teach `walk` to recognize bracketed segments, report templates separately from pages, and reject the bad names and stray files.
2. Extend `JSXParser`'s leading-statement matching to keep `params(...)` calls, and add a `params()` global that is a no-op at render time and an error outside a template.
3. Add the parameters pass to `ParseAll`: run each template's `params` against the real pages, validate and deduplicate the sets, and fill the URL pattern.
4. Register generated pages with their template source and seeded `params`, fill bare directories, add missing parent directories, and drop paths a real source owns.
5. Add generated rows to the route table, with the collision errors, so `build`, `serve`, and outputs pick them up.
6. Update the `web/` example site and the spec.

## Open issues

- Should `_layout.jsx`, `_defaults.jsx`, and `_outputs/` be allowed inside a bracketed directory, applying to every page generated under it? This proposal makes them errors and looks layouts up along the generated path, which covers the redesign.
- Should `pages()` take an option to leave generated pages out, instead of filtering on `params`? Filtering is one line, so this proposal adds nothing.
