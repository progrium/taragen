---
title: "How a site is laid out"
kind: manual
num: 4
order: 4
part: "building a site"
toc:
  - ["special-files", "Special files"]
  - ["page-paths-and-urls", "Page paths and URLs"]
  - ["folders-as-pages", "Folders as pages"]
---

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
[Generated pages](/docs/manual/generated-pages).

<a id="special-files"></a>

## Special files

<div class="tw"><table>
<thead><tr><th>File</th><th>Purpose</th></tr></thead>
<tbody>
<tr><td><code>_layout.jsx</code> / <code>_layout.tmpl</code></td><td class="prose">Wraps every page in its folder and below. See <a href="/docs/manual/layouts">Layouts</a>.</td></tr>
<tr><td><code>_globals.jsx</code></td><td class="prose">Values and helper functions available to all JSX in its folder and below. See <a href="/docs/manual/defaults-and-globals">Defaults and globals</a>.</td></tr>
<tr><td><code>_defaults.jsx</code></td><td class="prose">Default page data for pages in its folder and below.</td></tr>
<tr><td><code>_outputs/&lt;name&gt;.jsx</code></td><td class="prose">An extra output for every HTML page in its folder and below, such as a <code>.txt</code> version or a share card. See <a href="/docs/manual/extra-outputs-per-page">Extra outputs per page</a>.</td></tr>
<tr><td>Any other <code>_name.jsx</code> / <code>_name.tmpl</code></td><td class="prose">A partial, included with <code>partial(&#34;_dir/name&#34;)</code>. See <a href="/docs/manual/partials">Partials</a>.</td></tr>
<tr><td><code>_assets/</code> (root only)</td><td class="prose">Copied verbatim to <code>_public/_assets/</code>, never rendered.</td></tr>
<tr><td><code>_redirects</code> (root only)</td><td class="prose">Redirect rules in Netlify&#39;s format. See <a href="/docs/manual/aliases-and-redirects">Aliases and redirects</a>.</td></tr>
</tbody>
</table></div>

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

<a id="page-paths-and-urls"></a>

## Page paths and URLs

A page's URL comes from its file path: strip the extension, strip a trailing
`/index`, and the root becomes `/`.

<div class="tw"><table>
<thead><tr><th>Source</th><th>URL</th><th>Built file</th></tr></thead>
<tbody>
<tr><td><code>index.jsx</code></td><td class="prose"><code>/</code></td><td class="prose"><code>_public/index.html</code></td></tr>
<tr><td><code>about.md</code></td><td class="prose"><code>/about</code></td><td class="prose"><code>_public/about/index.html</code></td></tr>
<tr><td><code>about/index.jsx</code></td><td class="prose"><code>/about</code></td><td class="prose"><code>_public/about/index.html</code></td></tr>
<tr><td><code>articles/2021/ngrok.md</code></td><td class="prose"><code>/articles/2021/ngrok</code></td><td class="prose"><code>_public/articles/2021/ngrok/index.html</code></td></tr>
<tr><td><code>articles/feed.xml.jsx</code></td><td class="prose"><code>/articles/feed.xml</code></td><td class="prose"><code>_public/articles/feed.xml</code></td></tr>
</tbody>
</table></div>

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

<a id="folders-as-pages"></a>

## Folders as pages

Every folder is a page node, even without an index file. A folder with no index
file has no source and isn't built, but it still groups its children, so
`pages("articles")` lists everything in `articles/`, and `year.subpages` works
for a `2021/` folder with no index. These source-less nodes are called
**grouping nodes** in this manual.
