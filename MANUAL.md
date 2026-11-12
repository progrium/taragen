# Taragen manual

Taragen is a fast, minimal static site generator distributed as a single binary.
You write pages in JSX, Markdown, or Go templates, arrange them in folders, and
taragen turns the folder into a website. There is no config file, no Node.js, and
no `node_modules`: the file tree is the configuration, and JSX runs in a
JavaScript engine built into the binary.

Why taragen:

- **Write in the format that fits.** Markdown for writing, JSX for pages with
  logic, Go templates if you like them, all in the same site.
- **Components without a framework.** JSX layouts and partials let you build a
  site from reusable pieces, rendered to plain HTML with nothing shipped to the
  browser.
- **Live reloading.** `taragen serve` notices every change and refreshes the
  browser for you.
- **One binary.** Install it and you're done. Nothing to `npm install`, nothing
  to keep updated.

This manual is for people building sites with taragen. It covers every feature,
from a first page to generated archives, backlinks, redirects, and share cards,
and ends with [tutorials](#tutorials) that put them together.

## Contents

1. [Install](#install)
2. [Quick start](#quick-start)
3. [Commands](#commands)
4. [How a site is laid out](#how-a-site-is-laid-out)
5. [Pages](#pages)
6. [Page data](#page-data)
7. [Defaults and globals](#defaults-and-globals)
8. [Layouts](#layouts)
9. [Partials](#partials)
10. [Writing JSX](#writing-jsx)
11. [Markdown](#markdown)
12. [Go templates](#go-templates)
13. [Listing pages with `pages()`](#listing-pages-with-pages)
14. [Drafts](#drafts)
15. [Static files and assets](#static-files-and-assets)
16. [Non-HTML pages and feeds](#non-html-pages-and-feeds)
17. [Extra outputs per page](#extra-outputs-per-page)
18. [Share cards](#share-cards)
19. [Generated pages](#generated-pages)
20. [Backlinks](#backlinks)
21. [Aliases and redirects](#aliases-and-redirects)
22. [Code highlighting](#code-highlighting)
23. [Building and deploying](#building-and-deploying)
24. [Serve mode and live reload](#serve-mode-and-live-reload)
25. [Tutorials](#tutorials)
26. [Errors and troubleshooting](#errors-and-troubleshooting)
27. [Reference](#reference)

## Install

With Homebrew:

```sh
brew tap progrium/homebrew-taps
brew install taragen
```

Or download a release binary for your platform (macOS, Linux, and Windows, on
amd64 and arm64) from the GitHub releases page and put it on your `PATH`.

To build from source you need only Go:

```sh
make build   # writes .local/bin/taragen
make link    # symlinks it into /usr/local/bin
```

## Quick start

Make a folder for your site, and in it an `index.jsx`:

```jsx
<html>
  <body>
    <h1>Hello World</h1>
  </body>
</html>
```

From that folder, run:

```sh
taragen serve
```

Open `http://localhost:8088`. Edit the file and the browser reloads by itself.

Add a Markdown page, `about.md`:

```markdown
---
title: About
---

This site is built with **taragen**.
```

It appears at `/about`. Now give every page the same frame with a root
`_layout.jsx`:

```jsx
(content) =>
<html>
  <head>
    <meta charset="utf-8" />
    <title>{page.title ?? "My site"}</title>
  </head>
  <body>
    <nav><a href="/">Home</a> <a href="/about">About</a></nav>
    <main>{content}</main>
  </body>
</html>
```

Remove the `<html>` and `<body>` from `index.jsx`, since the layout provides
them now. When you're ready to publish:

```sh
taragen build
```

The finished site is in `_public/`, ready to upload to any static host.

## Commands

Both commands work on the current directory as the site root.

| Command | What it does |
| --- | --- |
| `taragen serve` | Serves the site at `http://localhost:8088`, renders pages on request, shows drafts, and live reloads the browser when files change. |
| `taragen build` | Cleans `_public/` and writes the whole site into it. Drafts are left out. Also available as `taragen gen` and `taragen generate`. |

The port (`8088`) and output directory (`_public`) are fixed.

`build` exits with a non-zero status on any error, so it is safe to use in CI
and deploy scripts.

## How a site is laid out

A site is a folder. Taragen walks it with a few simple rules:

- Files and folders whose names start with `.` are ignored.
- Files and folders whose names start with `_` are special: they are never
  pages and are never copied as-is (with the exception of the root `_assets/`
  folder). This is where layouts, partials, globals, defaults, output templates,
  and assets live.
- Files ending in `.jsx`, `.md`, or `.tmpl` are **page sources**.
- Folders are pages too (see [Folders as pages](#folders-as-pages)).
- Everything else is a **static file** and is copied to the same path in the
  output.

A path segment written as `[name]` is a page template parameter; see
[Generated pages](#generated-pages).

### Special files

| File | Purpose |
| --- | --- |
| `_layout.jsx` / `_layout.tmpl` | Wraps every page in its folder and below. See [Layouts](#layouts). |
| `_globals.jsx` | Values and helper functions available to all JSX in its folder and below. See [Defaults and globals](#defaults-and-globals). |
| `_defaults.jsx` | Default page data for pages in its folder and below. |
| `_outputs/<name>.jsx` | An extra output for every HTML page in its folder and below, such as a `.txt` version or a share card. See [Extra outputs per page](#extra-outputs-per-page). |
| Any other `_name.jsx` / `_name.tmpl` | A partial, included with `partial("_dir/name")`. See [Partials](#partials). |
| `_assets/` (root only) | Copied verbatim to `_public/_assets/`, never rendered. |
| `_redirects` (root only) | Redirect rules in Netlify's format. See [Aliases and redirects](#aliases-and-redirects). |

A typical site looks like this:

```
mysite/
├── _assets/
│   ├── fonts/Recursive.woff2
│   └── style.css
├── _globals.jsx
├── _defaults.jsx
├── _layout.jsx
├── _outputs/
│   ├── txt.jsx
│   └── card.png.jsx
├── _partials/
│   └── footer.jsx
├── _redirects
├── index.jsx
├── about.md
├── articles/
│   ├── _layout.jsx
│   ├── index.jsx
│   ├── feed.xml.jsx
│   └── 2021/
│       └── ngrok.md
└── tags/
    └── [tag].jsx
```

### Page paths and URLs

A page's URL comes from its file path: strip the extension, strip a trailing
`/index`, and the root becomes `/`.

| Source | URL | Built file |
| --- | --- | --- |
| `index.jsx` | `/` | `_public/index.html` |
| `about.md` | `/about` | `_public/about/index.html` |
| `about/index.jsx` | `/about` | `_public/about/index.html` |
| `articles/2021/ngrok.md` | `/articles/2021/ngrok` | `_public/articles/2021/ngrok/index.html` |
| `articles/feed.xml.jsx` | `/articles/feed.xml` | `_public/articles/feed.xml` |

Pages without an extension get clean URLs: they're written as
`<path>/index.html`. A page whose name keeps an extension after the format
extension, like `feed.xml.jsx`, is written as-is.

`about.jsx` and `about/index.jsx` are the same page. Use the folder form when a
page has files of its own, like images, next to it. Blog posts are just pages,
and a post with several images or downloads reads well as a folder:

```
articles/2021/ngrok/
├── index.md          the post, at /articles/2021/ngrok
├── diagram.png       linked from the post as diagram.png
└── tunnel.go
```

### Folders as pages

Every folder is a page node, even without an index file. A folder with no index
file has no source and isn't built, but it still groups its children, so
`pages("articles")` lists everything in `articles/`, and `year.subpages` works
for a `2021/` folder with no index. These source-less nodes are called
**grouping nodes** in this manual.

## Pages

A page source is a `.jsx`, `.md`, or `.tmpl` file. Each format produces an HTML
**body**, which is then wrapped by layouts.

| Format | Best for | Page data from |
| --- | --- | --- |
| JSX (`.jsx`) | Index pages, listings, anything with logic | A leading `data({...})` call |
| Markdown (`.md`) | Writing: articles, docs, notes | YAML frontmatter |
| Go template (`.tmpl`) | Simple templated pages and layouts | YAML frontmatter |

A JSX page:

```jsx
data({ title: "Articles" });
<section>
  <h1>{page.title}</h1>
  <ul>
    {pages("articles/**").reverse().map(a =>
      <li><a href={a.path}>{a.title}</a> <small>{a.date}</small></li>
    )}
  </ul>
</section>
```

A Markdown page:

```markdown
---
title: Building your own Ngrok in 130 lines
date: 2021-03-14
tags: [go, networking]
---

Ngrok is a tunnel...
```

## Page data

Every page has one data object, exposed as `page` in JSX and Go templates and
returned for each page by `pages()`. Frontmatter, `data()` calls, defaults, and
taragen's own keys all write into this same object.

### Where data comes from

Data is filled in this order, and later writers win:

1. Taragen seeds `path` and `slug` (and `params` for generated pages), and sets
   `outputs`.
2. `_defaults.jsx` files, from the site root down to the page's folder.
3. The page itself: YAML frontmatter in Markdown and templates, or `data({...})`
   in JSX.
4. Layouts, while rendering: frontmatter in template layouts, `data()` in JSX
   layouts.

After every page is parsed, taragen sets `links` and `backlinks` on every page.

### Built-in keys

Any key you set that isn't listed here is yours: `title`, `description`, `tags`,
and so on flow through untouched. These keys have built-in meaning:

| Key | Set by | Meaning |
| --- | --- | --- |
| `path` | taragen | The page's URL path, such as `/articles/2021/ngrok`. |
| `slug` | taragen | The last segment of the path, such as `ngrok`. |
| `src` | taragen | The raw page source (for generated pages, the template's source). |
| `body` | taragen | The page's rendered body, before layouts. |
| `isDir` | taragen | True for folders and `index.*` files. |
| `subpages` | taragen | In `pages()` results: the page's children, sorted, each with its own `subpages`. |
| `params` | taragen | Generated pages only: the parameter values, such as `{ year: "2026", month: "11" }`. |
| `outputs` | you, then taragen | You set it to turn outputs off; taragen replaces it with a map of output name to URL. See [Extra outputs per page](#extra-outputs-per-page). |
| `links` | taragen | Sorted list of the site pages this page's body links to. |
| `backlinks` | taragen | Pages whose bodies link to this page. See [Backlinks](#backlinks). |
| `aliases` | you | Old URLs that redirect to this page. Taragen replaces it with the normalized list. See [Aliases and redirects](#aliases-and-redirects). |
| `siteURL` | you | The site's absolute origin, such as `https://example.com`. Used to recognize absolute links to your own pages, and handy for building absolute URLs in feeds and meta tags. |
| `layout` | you | Layout path to use instead of the folder layouts, such as `_blog/_layout`. `""` turns layouts off. |
| `date` | you | `YYYY-MM-DD`. Orders listings; a future date makes the page a draft. |
| `draft` | you | `true` makes the page a draft. |
| `contentType` | you | The HTTP `Content-Type` in `serve`. Defaults to `text/html`. |
| `preamble` | you | Text written before the output. Defaults to `<!DOCTYPE html>` for HTML. |

Keys set by taragen replace whatever a page sets for them. Avoid using `outputs`,
`links`, `backlinks`, `aliases`, or `params` for your own data.

## Defaults and globals

Both are JSX files that run as plain scripts. Every bare assignment at the top
level becomes an export, so write `siteTitle = "My site"`, not `export` or
`const`:

```js
// _globals.jsx
siteTitle = "My site";
greeting = <strong>Hello!</strong>;
wrap = (text, font, width) => text.split(" ").reduce((lines, word) => {
  const last = lines[lines.length - 1];
  if (last && measureText(`${last} ${word}`, font) <= width) lines[lines.length - 1] = `${last} ${word}`;
  else lines.push(word);
  return lines;
}, []);
```

**`_globals.jsx`** defines names that every JSX file in its folder and below can
use: pages, layouts, partials, and output templates. Use it for site constants
and helper functions. Globals files run from the root down, so a nested
`_globals.jsx` sees and can override what its parents defined. Nested globals
are merged, so to hide a parent's global in a section, set it to `undefined` in
that section's `_globals.jsx`.

Since globals files are JSX, a global can be markup, like `greeting` above, and
used anywhere as `{greeting}`.

Globals are for JSX. Markdown and Go template pages can't read them directly;
to use one there, wrap it in a JSX partial and include the partial with
`[[ partial "_partials/name" ]]`.

**`_defaults.jsx`** sets default page data for every page in its folder and
below. Its exports are merged into each page's data before the page's own
frontmatter or `data()`, so pages can override them. `page` is available in it,
so defaults can depend on the page:

```js
// _defaults.jsx
siteURL = "https://example.com";

// articles/_defaults.jsx
aliases = [page.path.replace(/^\/articles/, "/blog")];
```

Built-in globals (`page`, `data`, `partial`, `raw`, `pages`, `params`,
`measureText`) are added after your globals, so they always win. Don't define
globals with those names.

Every JSX render runs in a fresh JavaScript VM, and globals are re-run for each
one. Pages can't leak state into each other, so don't try to share mutable state
through globals.

## Layouts

A layout wraps pages. A `_layout.jsx` is a function of one argument, the content
to wrap:

```jsx
(content) =>
<html>
  <head>
    <title>{page.title} · {siteTitle}</title>
  </head>
  <body>
    {partial("_partials/header")}
    <main>{content}</main>
    {partial("_partials/footer")}
  </body>
</html>
```

`content` is already-rendered HTML and is inserted as is. `page` is the data of
the page being wrapped.

### Nesting

A layout applies to pages in its folder and every folder below. Layouts nest
nearest first: a page in `articles/2021/` is wrapped by `articles/_layout.jsx`,
and the result is wrapped by the root `_layout.jsx`. Section layouts usually
return a fragment or a `<div>`, and leave `<html>` to the root layout:

```jsx
// articles/_layout.jsx
(content) =>
<article>
  <h1>{page.title}</h1>
  <time>{page.date}</time>
  {content}
</article>
```

### Choosing a layout explicitly

Set `layout` in a page's data to skip the folder layouts and use a specific one.
The value is a path from the site root, without the extension:

```markdown
---
layout: about/_wide
---
```

`layout: ""` turns layouts off entirely, which is what feeds and other non-HTML
pages want.

A layout can also set `layout` (with `data()` in JSX, or frontmatter in a
template layout). When it does, the remaining folder layouts are dropped and the
named layout is applied next. This lets one layout hand off to a specific chain:

```jsx
// about/_layout.jsx
data({ layout: "_layouts/wide" });
(content) => <div class="about">{content}</div>
```

Go template layouts (`_layout.tmpl`) get the content through `[[ content ]]`.

## Partials

Partials are reusable pieces of markup you can include in pages, layouts, and
other partials. Any `_name.jsx` or `_name.tmpl` file that isn't one of the
special files above is a partial, and so is any `.jsx` or `.tmpl` file inside an
`_`-prefixed folder. Most sites keep them together in a root `_partials/`
folder, where the files don't need their own underscore.

Include a partial from JSX with its path from the site root, without the
extension:

```jsx
{partial("_partials/footer")}
```

A navigation partial:

```jsx
// _partials/nav.jsx
<ul>
  <li><a href="/">Home</a></li>
  <li><a href="/articles">Articles</a></li>
  <li><a href="/about">About</a></li>
</ul>
```

A partial can be a plain element, as above, or a function. Arguments after the
name are passed to the function:

```jsx
// _partials/card.jsx
(post, showDate = true) =>
<div class="card">
  <a href={post.path}>{post.title}</a>
  {showDate && <time>{post.date}</time>}
</div>
```

```jsx
{pages("articles/**").map(p => partial("_partials/card", p))}
```

Partials see the same globals and `page` as the file that calls them. `.jsx` is
tried first, then `.tmpl`. A missing partial, or one that throws, fails the page
with an error naming it.

In Markdown and Go templates, use `[[ partial "_partials/footer" ]]`. Template partials
(`.tmpl`) currently receive no helper functions and ignore arguments, so prefer
JSX partials.

## Writing JSX

Taragen's JSX is a server-side template language. Elements turn straight into
HTML strings; there is no React, no components with state, no hooks, and nothing
runs in the browser. Write `class`, not `className`.

### Pages, data, and multiple roots

A JSX page can have several root elements, or start with an expression:

```jsx
data({ title: "Home" });
<h1>Welcome</h1>
<p>Latest posts:</p>
{partial("_partials/latest")}
```

A `data({...})` call at the very top of the file (after whitespace and comments)
sets page data. The trailing `;` is optional. `data()` merges its object into
both `page` and the page's data, so later code in the file and the layouts see
it. In a [generated page template](#generated-pages), a `params(...)` call can
sit alongside `data()` in these leading statements.

### Values in JSX

Children can be elements, strings, numbers, or arrays (flattened, so
`.map(...)` works). `null`, `undefined`, `true`, and `false` render nothing, so
conditionals work:

```jsx
{page.draft && <span class="badge">Draft</span>}
{page.tags?.map(t => <a href={`/tags/${t}`}>{t}</a>)}
```

Attribute values can be:

| Value | Result |
| --- | --- |
| string | `name="value"`; an empty string omits the attribute |
| number | Printed as JS prints it |
| `true` / `false` | Bare attribute (`<input disabled>`) / omitted |
| object | Serialized as a `style` string: `style={{ color: "red", "font-size": "2em" }}` |
| `null` / `undefined` | Omitted |

Anything else, such as an object as a child or a function as an attribute,
throws a `TypeError` and fails the render.

Attributes are written in the order you wrote them, and the output is identical
on every build, so diffs of `_public/` show only real changes. Void elements
like `<img>`, `<meta>`, and `<br>` are self-closed.

### Escaping and `raw()`

Text and attribute values are escaped, so a title with `&` or `"` is safe
anywhere, including in attributes and XML feeds. Text inside `<script>` and
`<style>` is written as is.

To insert HTML you trust, use `raw()`:

```jsx
<div class="post">{raw(post.body)}</div>
```

You need `raw()` for any HTML string, such as `body` from `pages()` results or
`page.body`. You don't need it for `partial()` results or a layout's `content`,
which are already HTML.

### Whitespace and `<pre>`

JSX normally collapses whitespace, but newlines inside `<pre>` blocks are kept,
so code samples written in JSX keep their lines.

### Output formatting

HTML output (`text/html`) is passed through an HTML formatter that normalizes
indentation. Other content types are written exactly as rendered.

## Markdown

Markdown pages start with optional YAML frontmatter between `---` lines, then
Markdown content. Raw HTML in Markdown is allowed and passed through.

Markdown is first run through the Go template engine, so `[[ ... ]]` actions
work in Markdown:

```markdown
---
title: Notes
---

This page lives at [[ page "path" ]].
```

This also means a literal `[[` anywhere in Markdown, even in a code block, is
read as a template action. Write it as `[[ "[[" ]]`, or use JSX for pages that
need a lot of it.

Fenced code blocks with a language are syntax highlighted; see
[Code highlighting](#code-highlighting).

## Go templates

`.tmpl` files use Go's `text/template` with `[[` and `]]` as delimiters, so they
don't clash with `{...}` in content and code. Output isn't auto-escaped.
Frontmatter works as in Markdown.

| Helper | Meaning |
| --- | --- |
| `[[ content ]]` | In a layout, the wrapped content; in a page, the body. |
| `[[ page "title" ]]` | A string value from page data. |
| `[[ partial "_partials/footer" ]]` | Includes a partial. |
| `safeHTML` | Returns its argument unchanged; kept for compatibility. |

The `page` helper only handles string values, and templates have no `pages`
helper, so lists like `tags`, `backlinks`, and listings are for JSX.

## Listing pages with `pages()`

`pages()` returns page data objects, which is how you build indexes, archives,
feeds, and tag pages. Each result has the page's full data, including `path`,
`title`, `date`, `body`, `outputs`, `links`, `backlinks`, and a recursive
`subpages` array.

### By path

A plain path returns the children of that page:

```jsx
pages("docs")          // pages directly inside docs/
```

### By pattern

A path containing wildcards returns the children of every page whose path
matches:

| Pattern | Returns |
| --- | --- |
| `articles/*` | Pages two levels below `articles` (children of each `articles/<x>`), e.g. posts in year folders. |
| `logs/202?` | Children of `logs/2020` through `logs/2029`. |
| `articles/**` | Every page below `articles`, at any depth. |
| `**` | Every page in the site except the root. |

`*`, `?`, and `[...]` match within one path segment and never cross a `/`. A
segment that is exactly `**` matches zero or more whole segments. (`a**` inside
a longer segment is just two `*`s.)

Patterns containing `**` leave out grouping nodes (folders with no index file),
since those only exist to hold the tree together. Plain paths, single-level
patterns, and `subpages` still include them.

Each page appears at most once in a result.

### The object form: several paths and filters

`pages()` also takes an object:

```jsx
pages({ path: ["articles/**", "logs/**", "docs/**"] })   // a mixed stream
pages({ where: { tags: "plan9" } })                         // everything tagged plan9
pages({ path: "articles/**", where: { featured: true } })
```

| Key | Default | Meaning |
| --- | --- | --- |
| `path` | `"**"` | A path or pattern as in the string form, or a list of them. A list returns the union, with no duplicates, sorted as one listing. |
| `where` | none | Page data keys and the values to keep. A page must match every key. |

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

### Order

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

### What's in a listing

- In `build`, drafts are left out of every result. In `serve`, they're
  included.
- [Generated pages](#generated-pages) are included, and are recognizable by
  their `params` key. To leave them out: `.filter(p => !p.params)`.
- Grouping nodes are included except in `**` patterns, as described above.

## Drafts

A page is a draft when its data has `draft: true`, or its `date` is in the
future. `build` skips drafts entirely: their pages, outputs, and aliases aren't
written, `pages()` doesn't return them, and links to and from them don't count
as backlinks. `serve` shows everything, so you can preview drafts locally.

Future-dated pages publish themselves: the first build on or after their date
includes them.

## Static files and assets

Any file that isn't a page source and isn't inside a `_` folder is copied to the
same path in `_public/`: `photos/cat.jpg` is served at `/photos/cat.jpg`. Files
next to a page in a page folder (`about/index.md`, `about/team.jpg`) can be
linked relatively.

The root `_assets/` folder is copied verbatim to `_public/_assets/`. Nothing in
it is ever rendered, even `.jsx` or `.md` files, which makes it the place for
CSS, scripts, fonts, and images:

```jsx
<link rel="stylesheet" href="/_assets/style.css" />
```

`_assets/` folders below the root are not copied. Other `_` folders are never
copied in `build`, though `serve` can still serve any file from disk.

## Non-HTML pages and feeds

A page can produce any text format. Keep the extra extension in the file name,
set its content type and preamble, and turn layouts off:

```jsx
// articles/feed.xml.jsx
data({
  preamble: '<?xml version="1.0" encoding="UTF-8" ?>',
  contentType: "application/rss+xml",
  layout: "",
});
<rss version="2.0">
  <channel>
    <title>{siteTitle}</title>
    <link>{page.siteURL}</link>
    {pages("articles/**").reverse().map(post =>
      <item>
        <title>{post.title}</title>
        <link>{`${page.siteURL}${post.path}`}</link>
        <guid>{`${page.siteURL}${post.path}`}</guid>
        <pubDate>{new Date(post.date).toUTCString()}</pubDate>
        <description>{raw(`<![CDATA[${post.body}]]>`)}</description>
      </item>
    )}
  </channel>
</rss>
```

This builds `_public/articles/feed.xml`. Because escaping is on, titles with `&`
produce valid XML.

Non-HTML pages don't get [extra outputs](#extra-outputs-per-page), aren't
counted as [backlink](#backlinks) sources, and their aliases serve a copy of the
content instead of a redirect stub.

## Extra outputs per page

An **output** is another file made from each page, next to it: a plain-text
version, a JSON summary, a share card image. Outputs are defined by templates in
an `_outputs/` folder.

### Defining an output

An output template is `_outputs/<name>.jsx`. The output's name is the first word
of the file name, and everything after the first `.` (before `.jsx`) is the URL
suffix:

| File | Name | Suffix | URL for `/articles/2021/ngrok` |
| --- | --- | --- | --- |
| `_outputs/txt.jsx` | `txt` | `.txt` | `/articles/2021/ngrok.txt` |
| `_outputs/json.jsx` | `json` | `.json` | `/articles/2021/ngrok.json` |
| `_outputs/card.png.jsx` | `card` | `.card.png` | `/articles/2021/ngrok.card.png` |

A name is a lowercase word of letters and digits, starting with a letter. The
root page uses `index`: `/index.txt`, `/index.card.png`.

`build` writes the output beside the page's folder:
`_public/articles/2021/ngrok.txt` sits next to
`_public/articles/2021/ngrok/index.html`. Static hosts serve both without any
setup.

A plain-text version of every page:

```jsx
// _outputs/txt.jsx
data({ contentType: "text/plain; charset=utf-8" });
`${page.title ?? page.date}\n\n${page.src}`
```

### What an output template sees

- `page` is the page's data, including `src` (the raw source) and `body` (the
  rendered body before layouts).
- All the page's globals, as for a layout.
- Layouts are not applied.

The template's last expression is the result. If it's a function, it's called
with no arguments. The result can be:

- a **string**, written exactly as is, with no escaping or formatting; or
- an **element**, serialized like a page.

Output templates are the only JSX files that may return a bare string.

`data()` in an output template sets the output's own settings, not the page's
data:

| Key | Meaning |
| --- | --- |
| `contentType` | The `Content-Type` in `serve`. Defaults to the type for the suffix's last extension (`text/plain; charset=utf-8` for `.txt`, `image/png` for `.png`), or `application/octet-stream` if unknown. |
| `preamble` | Text written before the output. Defaults to none, even for HTML. |
| `scale` | For PNG outputs only; see [Share cards](#share-cards). |

An output with `contentType: text/html` is formatted like an HTML page.

### Which pages get outputs

Every page that has a source and is `text/html` gets every output in scope.
Grouping nodes and non-HTML pages like `feed.xml.jsx` get none. A draft's
outputs are drafts too.

Output templates are scoped like layouts: one in `_outputs/` applies to pages in
that folder and below, and for each name, the nearest template wins. So
`logs/_outputs/card.png.jsx` replaces the root `_outputs/card.png.jsx` for pages
under `logs/`. If the nearer template has a different suffix, those pages get
the nearer suffix.

### Turning outputs off

Set `outputs` in a page's data:

```yaml
outputs: { card: false }   # no share card for this page
```

```yaml
outputs: false             # no outputs at all
```

Names that don't apply to the page are ignored. Since this is ordinary page
data, a `_defaults.jsx` can turn an output off for a whole section:

```js
// notes/_defaults.jsx
outputs = { txt: false };
```

### `page.outputs`

Taragen replaces `outputs` in every page's data with a map from output name to
URL, for the outputs that apply and weren't turned off:

```js
page.outputs // { txt: "/articles/2021/ngrok.txt", card: "/articles/2021/ngrok.card.png" }
```

The map is complete before any page renders, so layouts, page bodies, and
`pages()` results can all link to outputs:

```jsx
// in _layout.jsx
{page.outputs?.txt && <link rel="alternate" type="text/plain" href={page.outputs.txt} />}
```

Note that a page that turned an output off never sees its `false` in
`page.outputs`; the name is simply missing.

### Errors

An output template that throws fails the build, and in `serve` returns a 500
with the error. If an output's URL is the same as another page, output, static
file, alias, or redirect, the build fails naming both. For example, a static
`notes.txt` next to a page `notes` with a `txt` output is a collision.

## Share cards

A share card is the image social sites show for a link (`og:image`). In taragen
it's an ordinary output whose content type is `image/png`: the template returns
SVG, and taragen rasterizes it into a PNG using the site's own fonts. No
headless browser or external tool is involved.

### A card template

Name the template `_outputs/<name>.png.jsx` and return an `<svg>` with numeric
`width` and `height`:

```jsx
// _outputs/card.png.jsx
const W = 1200, H = 630, title = { family: "Recursive", size: 72, weight: 800 };
const lines = wrap(page.title ?? siteTitle, title, W - 160).slice(0, 4);
<svg width={W} height={H} viewBox={`0 0 ${W} ${H}`}>
  <rect width={W} height={H} fill="#fff" />
  <text x="80" y="104" font-family="Silkscreen" font-size="32">EXAMPLE.COM</text>
  <text font-family="Recursive" font-size={title.size} font-weight={title.weight}>
    {lines.map((line, i) => <tspan x="80" y={232 + i * 84}>{line}</tspan>)}
  </text>
  <rect x="80" y="520" width="1040" height="4" fill="#000" />
  <text x="80" y="574" font-family="Recursive" font-size="28">{page.date ?? ""}</text>
</svg>
```

(`wrap` is the helper shown in [Defaults and globals](#defaults-and-globals).)

Point your layout at it. `og:image` needs an absolute URL, so use `siteURL`:

```jsx
// in _layout.jsx
{page.outputs?.card && <>
  <meta property="og:image" content={`${page.siteURL}${page.outputs.card}`} />
  <meta property="og:image:width" content="1200" />
  <meta property="og:image:height" content="630" />
</>}
```

`build` writes `/articles/2021/ngrok.card.png` for every HTML page, and `serve`
renders each card when it's requested, so you can open a card URL in the browser
while designing it.

### Rules for card SVG

- Return a root `<svg>` element, or a string of SVG markup. The `xmlns`
  attribute is added if it's missing.
- The root needs numeric `width` and `height` in pixels, each at most 4096.
  These set the image size.
- `data({ scale: 2 })` renders at 2× (any whole number from 1 to 4) without
  changing your coordinates, for sharper cards on dense screens.
- Supported: shapes, paths, gradients, patterns, clipping, masks, filters,
  `<text>`, `<tspan>`, and `<image>`. Scripts, animation, and
  `<foreignObject>` are ignored. SVG has no text wrapping; measure and break
  lines yourself (below).
- Any output can be made a PNG output with
  `data({ contentType: "image/png" })`, whatever its suffix.

The same SVG and fonts always produce byte-identical PNGs.

### Fonts

Cards can use only fonts that are in your site. Every `.ttf`, `.otf`, `.ttc`,
`.woff`, and `.woff2` file anywhere in the site folder (except `_public/` and
dot-prefixed paths) is available, so a card can use the same font files your
CSS serves, such as `_assets/fonts/Recursive.woff2`. System fonts are never
used, so cards look the same on every machine.

Text picks a face by `font-family`, `font-weight`, and `font-style`, as in CSS.
If two files provide the same family, weight, and style, the one whose path sorts
first wins.

Every `font-family` must name at least one font your site has. Generic names
like `sans-serif` don't count unless you have a font with that name. An unknown
family fails the card with an error that lists the families taragen found, so a
typo shows up immediately instead of silently falling back.

### Measuring text with `measureText()`

`measureText(text, font)` returns the width of a string in pixels, using the
same fonts and shaping as the rasterizer:

```js
measureText("Building your own Ngrok", { family: "Recursive", size: 72, weight: 700 })
```

| Option | Default |
| --- | --- |
| `family` | required |
| `size` | `16` (pixels) |
| `weight` | `400` |
| `style` | `"normal"` |
| `letterSpacing` | `0` (pixels) |

An unknown family throws. `measureText` works in any JSX file, not just card
templates. Line breaking is up to you; the `wrap` helper above is a simple word
wrapper you can extend with ellipses or line limits.

### Images in cards

`<image href="...">` can use a root-relative path to a file in your site
(`/_assets/avatar.png`) or a `data:` URI. PNG, JPEG, GIF, WebP, and SVG are
supported. Remote URLs, `file:` URLs, relative paths, and paths that lead
outside the site folder are errors.

### Card errors

A card fails the build (or returns a 500 in `serve`) when the template doesn't
return SVG, `width` or `height` is missing or over 4096, a font family is
unknown, or an image reference isn't allowed.

## Generated pages

Some pages don't have a source file of their own: one page per tag, one archive
per month. A **page template** generates many pages from one file.

### Bracketed names

Write a path segment as `[name]` to make it a parameter. The source is a template
if its file name or any folder above it is a parameter:

| Source | URL pattern | Example URLs |
| --- | --- | --- |
| `tags/[tag].jsx` | `/tags/[tag]` | `/tags/go`, `/tags/plan9` |
| `logs/[year]/index.jsx` | `/logs/[year]` | `/logs/2025`, `/logs/2026` |
| `logs/[year]/[month].jsx` | `/logs/[year]/[month]` | `/logs/2026/11` |
| `tags/[tag]/feed.xml.jsx` | `/tags/[tag]/feed.xml` | `/tags/go/feed.xml` |

Names are lowercase letters and digits, starting with a letter, and each name
appears at most once in a path. The whole segment must be the parameter:
`[year]-archive.jsx` is an error, not a page.

Templates must be JSX. A Markdown or Go template file inside a bracketed path is
an error, and so is a static file or `_`-prefixed file inside a bracketed folder.
Put layouts, defaults, and output templates for generated pages in a real folder
above the bracketed one (see [Layouts and data](#layouts-and-data-for-generated-pages)).

### Declaring the pages with `params()`

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

### What `params()` can see

`params()` runs before any generated page exists, in its own VM with your
globals and `pages()`. `page` is undefined there. `pages()` returns only real
pages, never generated ones, so one template can't feed another and the set of
pages depends only on the files on disk.

Once all pages exist, page bodies and layouts see everything: a "Tags" index
page can call `pages("tags")` and list every generated tag page.

### Generated page data

A generated page's data is built like any page's, plus `params`:

| Key | Value for `/logs/2026/11` |
| --- | --- |
| `params` | `{ year: "2026", month: "11" }` (always strings) |
| `path` | `/logs/2026/11` |
| `slug` | `11` |
| `src` | The template's source |
| `isDir` | True if the template is an `index.jsx` |

### Layouts and data for generated pages

Globals, defaults, layouts, and output templates are found by the generated
page's path, exactly as if a real file lived there. `logs/_layout.jsx` wraps
`/logs/2026/11`, and a real `logs/2026/_defaults.jsx` applies to it. HTML
generated pages get outputs from `_outputs/` like any page, so
`/logs/2026/11.txt` exists if `_outputs/txt.jsx` does.

If a generated page's parent folder doesn't exist as a page, taragen adds a
grouping node for it, so `pages("logs/*")` reaches years that exist only through
generated pages.

### Overriding a generated page

A real source at the same path wins, silently. To give one tag a hand-written
page with its own intro, add `tags/plan9.md`; every other tag keeps its generated
page. A generated page at the path of a grouping node fills it, keeping its
subpages: that's how `logs/[year]/index.jsx` gives `/logs/2026` an index.

Two templates producing the same path is an error, as is a generated page whose
URL matches a static file, output, alias, or redirect.

### Generated pages in listings

Generated pages show up in `pages()` results, including `**` patterns. A home
stream built from `pages("logs/**")` will include generated month pages; skip
them with a filter on `params`:

```jsx
pages({ path: ["articles/**", "logs/**"] }).filter(p => !p.params).reverse()
```

### Drafts and serving

`params()` follows the draft rules: in `build` its `pages()` calls never see
drafts, so a month that only holds a draft gets no page; in `serve` it does. A
generated page whose own data makes it a draft is skipped by `build`.

In `serve`, a URL that fits the pattern but wasn't generated, like
`/tags/nosuchtag`, isn't a page. Adding a tag to a Markdown file creates its page
on the next reload.

## Backlinks

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

### `links` and `backlinks`

Every page gets two lists, empty when there's nothing:

- `page.links`: the sorted URLs of site pages this page's body links to, such as
  `["/docs/outputs", "/logs/2026/ngrok"]`.
- `page.backlinks`: the pages that link to this page, in the usual
  [listing order](#order). Each entry is a copy of the linking page's data, like
  a `pages()` result, but without `links`, `backlinks`, or `subpages` (so there
  are no cycles). To get more about a referrer, call `pages()` on its path.

`pages()` results carry both lists too.

### What counts as a link

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

### How hrefs are resolved

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

### When backlinks are available

The link graph is computed once per full parse of the site: in `build`, and in
`serve` after each change. Index pages that call `pages()` are parsed a second
time, once all pages exist, so their links to everything they list count.

Layouts always see the full graph, as does any page body in the final output.
Links rendered from `backlinks` themselves (a "linked from" list) are not
collected, so a backlinks section never feeds back into the graph.

One side effect: a page body that calls `pages()` runs twice per parse, so
something like `console.log` in it prints twice.

## Aliases and redirects

When a page moves, keep its old URL working with an **alias**. For redirects
that aren't tied to a page, use a **`_redirects`** file.

### Page aliases

List old URLs in `aliases`:

```yaml
---
title: Building your own Ngrok in 130 lines
aliases: [/blog/2021/building-your-own-ngrok-in-130-lines]
---
```

A single string is fine too: `aliases: /old-url`.

Because it's page data, one line in a `_defaults.jsx` covers a whole section.
After moving `blog/` to `articles/`:

```js
// articles/_defaults.jsx
aliases = [page.path.replace(/^\/articles/, "/blog")];
```

Now `/articles/2021/ngrok` has the alias `/blog/2021/ngrok`, and `/articles`
has `/blog`. A page that sets its own `aliases` replaces the default rather than
adding to it.

Aliases are read from defaults, frontmatter, and the page's `data()`. Aliases
set by a layout are ignored. Only built pages get aliases (not grouping nodes),
and a draft's aliases are drafts too.

Alias rules:

- Each alias must start with `/`. A trailing `/` is removed.
- It can't contain `?`, `#`, `\`, `//`, or a `.` or `..` segment.
- Invalid aliases are errors naming the page. Duplicates are dropped.

After this, `page.aliases` is the cleaned-up list, which layouts and `pages()`
can read, for example to list a page's old URLs.

### What an alias produces

For an HTML page, `build` writes a small redirect page at each alias
(`/blog/2021/ngrok/index.html`, or as-is for an alias with an extension like
`/old.html`):

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

These stubs work on any static host, including ones like GitHub Pages that
can't send real redirects. They don't use layouts or globals, and they're
identical on every build. The canonical link uses the page's path, not an
absolute URL, and a `#fragment` in the old URL isn't carried over.

`serve` answers an alias with a real `301 Moved Permanently`.

For a non-HTML page, such as a feed, a redirect page would break feed readers,
so the alias serves a **copy** of the page's output with its content type, in
both `build` and `serve`. Old feed URLs keep working for existing subscribers.

### The `_redirects` file

For other redirects, add a `_redirects` file at the site root, in a subset of
Netlify's format:

```
# from            to                               status
/newsletter       https://buttondown.com/progrium  302
/blog/*           /articles/:splat
/talks            /articles?tag=talks
```

Each line is `from to [status]`. Blank lines are ignored and `#` starts a
comment line.

- `from` is a path, cleaned up like an alias. It may end in `/*` to match
  everything below it; `*` can't appear anywhere else.
- `to` is a path starting with `/`, or an absolute `http`/`https` URL. With a
  wildcard `from`, `:splat` in `to` is replaced by what the `*` matched. A path
  `to` may include a query string.
- `status` is `301`, `302`, `307`, or `308`. The default is `301`.

Other Netlify features are errors naming the line: other statuses (`200`
rewrites, `404`), the `!` force suffix, named placeholders like `:year`, query
conditions, and extra fields.

`build` copies `_redirects` to `_public/_redirects`, so hosts that read it
(Netlify, Cloudflare Pages, and others) send real redirects. It also writes
redirect pages from the rules, so hosts that don't read it still work:

- **A rule without a wildcard** writes one redirect page at `from`. If `to` is
  one of your non-HTML pages, it writes a copy of that page instead.
- **A wildcard rule pointing at a path with `:splat`** is expanded against your
  HTML pages. For `/blog/* /articles/:splat`, every built page under
  `/articles/` gets a redirect page at the matching `/blog/` path. Drafts aren't
  expanded.
- **Other wildcard rules**, such as ones pointing off-site or without `:splat`,
  can't be expanded. `build` prints a warning naming the line: the rule only
  works on hosts that read `_redirects`.
- A rule whose `from` has an extension other than `.html` gets no redirect page,
  and a warning, unless its `to` is one of your non-HTML pages, since an HTML
  page there would be served with the wrong type.

In `serve`, every rule works with its real status, including wildcard rules
`build` couldn't expand. The request's query string is kept when `to` has none.

### Conflicts

Aliases and redirect rules share one URL space with pages, outputs, and static
files. These fail the build, naming both sides:

- an alias equal to a page's own path, or to another page's alias;
- an alias that is also the `from` of a rule in `_redirects`;
- an alias or rule that lands on a page, output, or static file.

Paths expanded from a wildcard rule are the exception: if something already
lives at the path, the rule skips it, and between overlapping wildcard rules the
first one wins. So you can keep a real page at `/blog/2021/something` under a
`/blog/*` rule, as on Netlify.

In `serve`, a request is matched against pages, outputs, aliases, and plain
rules first, then wildcard rules in file order, then files on disk.

## Code highlighting

Fenced code blocks in Markdown with a language are highlighted using CSS
classes, so your stylesheet controls how code looks:

````markdown
```go
func main() {}
```
````

renders as:

```html
<pre class="chroma" tabindex="0"><code class="language-go"><span class="line"><span class="cl"><span class="kd">func</span> <span class="nf">main</span><span class="p">()</span> <span class="p">{}</span>
</span></span></code></pre>
```

- The `<pre>` has `class="chroma"` and no inline styles. `tabindex="0"` lets
  keyboard users scroll wide blocks.
- The `<code>` has `class="language-<lang>"`, using the first word of the fence
  as written.
- Each line is `<span class="line"><span class="cl">...</span></span>`, and
  each token is a `<span>` with Chroma's short class name for its type. Plain
  text gets no span.

Common classes:

| Class | Token |
| --- | --- |
| `k` | Keyword |
| `kd` | Declaration keyword (`func`, `var`) |
| `nf` | Function name |
| `nx` | Other name |
| `s`, `s1`, `s2` | Strings |
| `m`, `mi` | Numbers |
| `c`, `c1`, `cm` | Comments |
| `o` | Operator |
| `p` | Punctuation |

The full list is in Chroma's token type table
(<https://github.com/alecthomas/chroma/blob/master/types.go>). The markup is
the same as Hugo's, so any Chroma stylesheet works: generate one with Chroma's
CLI or `hugo gen chromastyles` and paste it into your CSS.

Taragen adds no highlighting CSS. Without any, code shows in your normal `pre`
and `code` styles. A minimal monochrome style:

```css
.chroma [class^="k"] { font-weight: bold; }                /* keywords */
.chroma [class^="c"]:not(.cl) { font-style: italic; }      /* comments */
```

Dark mode is just your usual `prefers-color-scheme` rules.

Blocks in a language Chroma doesn't know, blocks without a language, indented
code blocks, and inline code are plain `<pre><code>` (keeping
`class="language-<lang>"` when a language was given). Taragen never guesses a
language.

## Building and deploying

`taragen build` writes the site to `_public/`:

1. **Clean.** Everything in `_public/` is removed except dot-prefixed entries at
   its top level. A deploy repository checked out at `_public/.git` survives,
   while renamed and deleted pages don't linger.
2. **Assets.** The root `_assets/` is copied to `_public/_assets/`.
3. **Everything else.** Each URL the site has is written once: pages, outputs
   (including PNG cards), generated pages, alias and redirect pages, static
   files, and `_redirects`.

Drafts are skipped throughout.

Any error stops the build with a non-zero exit: a JSX error, a missing partial,
a bad `params()` set, an invalid alias or `_redirects` line, a card with an
unknown font, or two things claiming the same URL. Warnings, such as a wildcard
redirect that can't be expanded, are printed but don't fail the build.

Builds are deterministic: the same sources produce the same bytes, including
attribute order, listing order, and PNG cards. That makes `_public/` diffs
reviewable.

To deploy, upload `_public/` to any static host. Clean URLs
(`/about` → `/about/index.html`) and files beside folders
(`/about.txt` next to `/about/`) work on GitHub Pages, Netlify, S3, and similar
hosts without configuration.

## Serve mode and live reload

`taragen serve` is for writing:

- Pages, outputs, generated pages, and cards render when requested, from the
  latest source.
- Drafts are visible and included in listings.
- Aliases answer with `301`; `_redirects` rules answer with their status.
- HTML responses get a small script that reloads the browser when anything in
  the site changes: pages, CSS, images, fonts, `_redirects`, anything outside
  dot-prefixed paths and `_public/`. Non-HTML responses (text outputs, cards,
  feeds) don't get the script.
- A render error shows as a 500 with the error text.
- Paths that aren't pages, outputs, aliases, or redirects are served from disk,
  including underscore paths like `/_assets/...`.

After a change to a page source, a folder, `_outputs/`, a font, or
`_redirects`, the whole site is re-read, so new pages, generated pages,
backlinks, and aliases appear on the next reload.

Requests are rendered one at a time. That's fine for local preview; use
`build` for production.

## Tutorials

These walk through common tasks from start to finish. Each one links back to
the sections above for the details.

### Set up a blog

Posts are ordinary pages. Grouping them in year folders keeps the tree tidy and
gives you a natural archive.

1. Create an `articles/` folder with a folder for the current year, and a post
   in it, `articles/2025/hello-world.md`:

   ```markdown
   ---
   title: Hello, world
   date: 2025-01-01
   description: The first post.
   ---

   This is my first post.
   ```

   The `date` orders posts in listings, and a date in the future keeps a post
   out of `build` until that day (see [Drafts](#drafts)).

2. Give posts a frame with `articles/_layout.jsx`. It wraps every page in
   `articles/`, and the root layout wraps the result:

   ```jsx
   (content) =>
   <article class="post">
     <h1>{page.title}</h1>
     <time>{page.date}</time>
     {content}
   </article>
   ```

3. Add the blog's front page, `articles/index.jsx`, listing posts by year,
   newest first. Year folders have no index file of their own, so they're
   grouping nodes with `isDir` set and their posts in `subpages`:

   ```jsx
   data({ title: "Articles", layout: "_layout" });
   <h1>Articles</h1>
   {pages("articles").filter(p => p.isDir).reverse().map(year =>
     <section>
       <h2>{year.slug}</h2>
       <ul>
         {year.subpages.slice().reverse().map(post =>
           <li><time>{post.date}</time> <a href={post.path}>{post.title}</a></li>
         )}
       </ul>
     </section>
   )}
   ```

   Setting `layout: "_layout"` uses only the root layout for this page, since
   the post layout would add a stray date heading.

   For a flat list instead, one call reaches every post however it's nested:

   ```jsx
   {pages("articles/**").reverse().map(post => partial("_partials/card", post))}
   ```

4. Add a feed. `articles/feed.xml.jsx` builds `/articles/feed.xml`; the full
   template is in [Non-HTML pages and feeds](#non-html-pages-and-feeds). Set
   `siteURL` in your root `_defaults.jsx` so the feed can build absolute links,
   and advertise the feed from the root layout's `<head>`:

   ```jsx
   <link rel="alternate" type="application/rss+xml" title="Articles" href="/articles/feed.xml" />
   ```

5. Run `taragen serve` and open `/articles`.

### Add tag pages

Tag posts in frontmatter with `tags: [go, networking]`. Then one template makes
a page for every tag in use, `tags/[tag].jsx`:

```jsx
params(() => [...new Set(pages("articles/**").flatMap(p => p.tags ?? []))].map(tag => ({ tag })));
data({ title: `Tagged “${page.params.tag}”` });
<>
  <h1>{page.title}</h1>
  <ul>
    {pages({ path: "articles/**", where: { tags: page.params.tag } }).reverse().map(p =>
      <li><a href={p.path}>{p.title}</a></li>
    )}
  </ul>
</>
```

A tag index, `tags/index.jsx`, lists the generated pages, since page bodies run
after every page exists:

```jsx
data({ title: "Tags" });
<ul>{pages("tags").map(t => <li><a href={t.path}>{t.slug}</a></li>)}</ul>
```

Link each post's tags from the post layout:

```jsx
{page.tags?.map(t => <a class="tag" href={`/tags/${t}`}>{t}</a>)}
```

To give one tag a hand-written page, add `tags/go.md`; it replaces the
generated `/tags/go` and every other tag keeps its generated page. See
[Generated pages](#generated-pages).

### Add yearly archive pages

The year folders from the blog tutorial have no page of their own, so `/articles/2025`
is a 404. A template named `index.jsx` inside a bracketed folder fills them in,
`articles/[year]/index.jsx`:

```jsx
params(() => pages("articles/**").filter(p => p.date).map(p => ({ year: p.date.slice(0, 4) })));
data({ title: `Articles from ${page.params.year}` });
<ul>
  {pages(`articles/${page.params.year}`).reverse().map(p =>
    <li><a href={p.path}>{p.title}</a></li>
  )}
</ul>
```

Each generated year page takes over the year's grouping node and keeps its
posts as `subpages`. Because these pages now have a source, `pages("articles/**")`
includes them; streams that want only posts add `.filter(p => !p.params)`.

### Build a docs section with a sidebar

Put docs pages in `docs/`, with subsections as folders, and give the section a
layout that lists its pages:

```jsx
// docs/_layout.jsx
(content) =>
<>
  <nav class="sidebar">
    <ul>
      {pages("docs").map(p =>
        <li>
          <a href={p.path}>{p.title}</a>
          {p.isDir && p.subpages.length > 0 &&
            <ul>{p.subpages.map(sub => <li><a href={sub.path}>{sub.title}</a></li>)}</ul>}
        </li>
      )}
    </ul>
  </nav>
  <section>
    <h1>{page.title}</h1>
    {content}
  </section>
</>
```

Pages without a `date` sort by slug, so to control the order, prefix file names
(`01-install.md`) or sort in the layout by a key of your own, such as `order`
in frontmatter:

```jsx
pages("docs").sort((a, b) => (a.order ?? 99) - (b.order ?? 99))
```

A subsection index, such as `docs/tutorials/index.jsx`, lists its own pages:

```jsx
data({ title: "Tutorials" });
<ul>{pages("docs/tutorials").map(p => <li><a href={p.path}>{p.title}</a></li>)}</ul>
```

If `docs/` has no index page, send `/docs` straight to the first page with a
line in `_redirects`, instead of writing a redirect page by hand:

```
/docs   /docs/getting-started   302
```

### Highlight the current page in navigation

`page` in a partial is the page being rendered, so a nav partial can compare
each link with `page.path`:

```jsx
// _partials/nav.jsx
() => {
  const link = (href, text) =>
    <a class={page.path === href || page.path.startsWith(href + "/") ? "selected" : ""} href={href}>{text}</a>;
  return (
    <ul>
      <li>{link("/", "Home")}</li>
      <li>{link("/articles", "Articles")}</li>
      <li>{link("/about", "About")}</li>
    </ul>
  );
}
```

The `startsWith` check keeps "Articles" selected on every post. An empty class
is left out of the output, so unselected links have no `class` attribute. Style
it in your CSS:

```css
nav .selected { font-weight: bold; }
```

Include it in a layout with `{partial("_partials/nav")}`.

### Format dates

`date` is a `YYYY-MM-DD` string. Add a formatter to your root `_globals.jsx`:

```js
formatDate = (date, format = "MM/dd/yyyy") => {
  const d = new Date(date);
  const map = {
    yyyy: String(d.getUTCFullYear()),
    MM: String(d.getUTCMonth() + 1).padStart(2, "0"),
    dd: String(d.getUTCDate()).padStart(2, "0"),
  };
  return format.replace(/yyyy|MM|dd/g, m => map[m]);
};
```

and use it in any JSX:

```jsx
<time datetime={page.date}>{formatDate(page.date, "dd.MM.yyyy")}</time>
```

The UTC methods matter: `new Date("2025-01-01")` is midnight UTC, and local
time methods would show the previous day west of Greenwich.

To use a global in a Markdown page, wrap it in a partial, such as
`_partials/today.jsx` containing `<span>{formatDate(new Date())}</span>`, and
include it with `[[ partial "_partials/today" ]]`.

### Offer a plain-text version of every page

Add `_outputs/txt.jsx`:

```jsx
data({ contentType: "text/plain; charset=utf-8" });
`${page.title ?? page.date}\n\n${page.src}`
```

Every HTML page now has a `.txt` twin, `/about` beside `/about.txt`, holding
its title and raw source. Link it from the root layout:

```jsx
{page.outputs?.txt && <a href={page.outputs.txt}>Plain text</a>}
```

Turn it off where it makes no sense with `outputs: { txt: false }` in a page's
frontmatter, or for a whole folder in its `_defaults.jsx`. See
[Extra outputs per page](#extra-outputs-per-page).

### Add share cards

1. Put the font files you want on cards in the site, for example
   `_assets/fonts/Inter-Bold.ttf`. The same files can serve your CSS.
2. Add the `wrap` helper from [Defaults and globals](#defaults-and-globals) to
   `_globals.jsx`.
3. Create `_outputs/card.png.jsx`, returning a 1200×630 `<svg>` like the
   example in [Share cards](#share-cards), with `font-family="Inter"`.
4. Point the root layout's `<head>` at it:

   ```jsx
   {page.outputs?.card && <meta property="og:image" content={`${page.siteURL}${page.outputs.card}`} />}
   ```

5. With `taragen serve` running, open `/about.card.png` in the browser. Each
   save reloads it, so you can tune the layout by eye.

If a page should share a different image, set `outputs: { card: false }` and
write your own `og:image` tag for it.

### Show what links to a page

Add a "Linked from" list to the post layout:

```jsx
{page.backlinks.length > 0 && <aside>
  <h2>Linked from</h2>
  <ul>{page.backlinks.map(p => <li><a href={p.path}>{p.title}</a></li>)}</ul>
</aside>}
```

Only links in page bodies count, not nav or footer links, so the list shows
real references. Set `siteURL` if your posts link to each other with full
URLs. See [Backlinks](#backlinks).

### Move a section without breaking links

To rename `blog/` to `articles/`:

1. Move the folder.
2. Add `articles/_defaults.jsx` so every page answers at its old URL too:

   ```js
   aliases = [page.path.replace(/^\/articles/, "/blog")];
   ```

3. Build. `/blog/2021/ngrok` gets a redirect page to `/articles/2021/ngrok`,
   `/blog` redirects to `/articles`, and the old feed URL `/blog/feed.xml` keeps
   serving the feed itself, so subscribers don't notice. Old pages are gone from
   `_public/` because every build starts clean.

On a host that reads `_redirects`, you can add `/blog/* /articles/:splat` too,
for real 301s. See [Aliases and redirects](#aliases-and-redirects).

### Style code blocks

Fenced code in Markdown comes out with Chroma classes and no colors. Generate a
stylesheet for a theme you like:

```sh
hugo gen chromastyles --style=monokai > _assets/chroma.css
```

or with Chroma's own CLI, and link it from your layout. Or write a few rules
yourself; see [Code highlighting](#code-highlighting).

## Errors and troubleshooting

| Symptom | Cause and fix |
| --- | --- |
| Page shows `&amp;` or `&lt;p&gt;` where HTML should be | You inserted an HTML string as text. Wrap it in `raw()`: `{raw(post.body)}`. |
| `TypeError` from JSX | A child or attribute is an unsupported value, like an object as a child. Render a field of it instead. |
| "partial not found" | The path in `partial()` is from the site root and without extension: `partial("_partials/footer")`. |
| A page appears in `serve` but not in `build` | It's a draft: `draft: true` or a future `date`. |
| `[[` in a Markdown page breaks it | Markdown runs through the template engine first. Write it as `[[ "[[" ]]`, or use JSX for that page. |
| URL collision error | Two things produce one URL (a page and static file, an output and a page, two aliases). The error names both; rename or remove one. |
| A tag page isn't generated | `params()` only sees real, non-draft pages (in `build`). Check the source page has the tag and isn't a draft. |
| A card fails with an unknown font family | Add the font file to the site (e.g. `_assets/fonts/`) and use its family name exactly. System fonts aren't used. |
| A link isn't in `backlinks` | Links in layouts don't count; links to aliases, outputs, and static files don't count; absolute links need `siteURL`. |
| A relative link counts toward the wrong page | Relative links resolve against `<path>/` for clean-URL pages, as in the browser. Prefer root-relative links (`/docs/setup`). |
| A wildcard redirect works in `serve` but not on the host | It couldn't be expanded into redirect pages (see the build warning). Use a host that reads `_redirects`, or point it at a path with `:splat`. |
| Code isn't colored | Taragen emits classes, not colors. Add Chroma CSS to your stylesheet. |
| Logs print twice | Bodies that call `pages()` are parsed twice so listings see every page. |

## Reference

### Built-in JSX globals

| Global | Available in | Description |
| --- | --- | --- |
| `page` | Pages, layouts, partials, outputs, defaults | The current page's data. |
| `data(obj)` | Pages, layouts, outputs | Merge `obj` into page data (in outputs: the output's settings). |
| `partial(name, ...args)` | All JSX | Render a partial and insert its HTML. |
| `raw(html)` | All JSX | Insert an HTML string unescaped. |
| `pages(query)` | All JSX | List pages by path, pattern, or `{ path, where }`. |
| `params(sets)` | Page templates, as a leading statement | Declare the pages a template generates. |
| `measureText(text, font)` | All JSX | Width of `text` in pixels, using the site's fonts. |

### Output template `data()` keys

| Key | Default |
| --- | --- |
| `contentType` | From the suffix's last extension, else `application/octet-stream` |
| `preamble` | None |
| `scale` | `1` (PNG outputs; whole number up to 4) |

### `_redirects` syntax

```
# comment
from        to                 [301|302|307|308]
/old        /new
/old/*      /new/:splat
/ext        https://example.com/page   302
```

### File naming summary

| Pattern | Meaning |
| --- | --- |
| `.name` | Ignored |
| `_name/` | Special folder, never a page or copied (except root `_assets/`) |
| `_layout.jsx` / `_layout.tmpl` | Layout |
| `_globals.jsx` | Globals |
| `_defaults.jsx` | Default page data |
| `_outputs/<name>[.ext].jsx` | Output template |
| `_name.jsx` / `_name.tmpl` | Partial |
| `_redirects` (root) | Redirect rules |
| `name.jsx`, `name.md`, `name.tmpl` | Page at `/name` |
| `name/index.*` | Page at `/name` |
| `name.xml.jsx` | Page at `/name.xml` |
| `[param].jsx`, `[param]/` | Page template |
| anything else | Static file |
