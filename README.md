# taragen

Fast, minimal static site generator.

Taragen turns a folder of JSX, Markdown, and Go template files into a static
website. It ships as a single binary with a JavaScript engine built in, so there
is no Node.js, no `npm install`, and no config file: the file tree is the
configuration.

```sh
brew tap progrium/homebrew-taps
brew install taragen

mkdir mysite && cd mysite
echo '<h1>Hello World</h1>' > index.jsx
taragen serve     # preview at http://localhost:8088 with live reload
taragen build     # write the site to _public/
```

**Read the [manual](MANUAL.md)** for a full guide to building sites, with
[tutorials](MANUAL.md#tutorials) for blogs, tag pages, docs sections, share
cards, and more.

## Features

**Pages and templates**

- JSX pages, layouts, and partials, rendered to HTML on the server with no
  client runtime. Text and attributes are escaped by default, with `raw()` for
  trusted HTML.
- Markdown pages with YAML frontmatter, and Go templates (`[[ ]]` delimiters).
- Layouts that nest by folder, and an explicit `layout` key to choose or turn
  them off.
- Global JS helpers (`_globals.jsx`) and default page data (`_defaults.jsx`),
  scoped to their folder and below.
- Any text format as a page: RSS feeds, sitemaps, JSON, with their own content
  type.

**Content and navigation**

- `pages()` listings by path, wildcard, or recursive `**` pattern, filtered by
  page data with `where`, in a stable order on every build.
- Generated pages from bracketed file names, such as `tags/[tag].jsx` or
  `logs/[year]/[month].jsx`, for tag pages and archives.
- Backlinks: every page knows which pages link to it, from links in page
  bodies.
- Drafts, by flag or future date, visible in `serve` and left out of `build`.

**Outputs**

- Extra outputs per page from `_outputs/` templates, like a `.txt` version
  beside every page, listed in `page.outputs` for layouts and feeds.
- PNG share cards rendered from SVG templates with your site's own fonts, plus
  `measureText()` for fitting titles. No headless browser.
- Aliases for moved pages and a Netlify-style `_redirects` file, written as
  redirect pages that work on any static host and served as real redirects in
  development.
- Syntax highlighting with Chroma CSS classes, styled by your stylesheet.

**Workflow**

- `taragen serve` renders on request, shows drafts, and live reloads on any
  change, including CSS.
- `taragen build` cleans `_public/` (keeping a `.git` deploy checkout), and
  produces byte-identical output for the same sources, so diffs are reviewable.
- URL collisions between pages, outputs, aliases, and files are build errors,
  never silent overwrites.
- One binary for macOS, Linux, and Windows on amd64 and arm64.

## A taste

A site:

```
mysite/
├── _assets/style.css
├── _defaults.jsx          siteURL = "https://example.com"
├── _globals.jsx           siteTitle = "My site"
├── _layout.jsx
├── _outputs/
│   └── card.png.jsx       one share card per page
├── index.jsx
├── articles/
│   ├── index.jsx
│   ├── feed.xml.jsx
│   └── 2021/ngrok.md
└── tags/[tag].jsx         one page per tag
```

A layout, using backlinks and the share card output:

```jsx
// _layout.jsx
(content) =>
<html>
  <head>
    <title>{page.title} · {siteTitle}</title>
    <link rel="stylesheet" href="/_assets/style.css" />
    {page.outputs?.card && <meta property="og:image" content={`${page.siteURL}${page.outputs.card}`} />}
  </head>
  <body>
    <main>{content}</main>
    {page.backlinks.length > 0 && <aside>
      <h2>Linked from</h2>
      <ul>{page.backlinks.map(p => <li><a href={p.path}>{p.title}</a></li>)}</ul>
    </aside>}
  </body>
</html>
```

A page per tag:

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

The [manual](MANUAL.md) covers each of these in detail.

## Install

- **Homebrew:** `brew tap progrium/homebrew-taps && brew install taragen`
- **Binary:** download a release for your platform from
  [GitHub releases](https://github.com/progrium/taragen/releases).
- **From source:** with Go installed, `make build` writes
  `.local/bin/taragen`, and `make link` symlinks it into `/usr/local/bin`.

## Documentation

- [MANUAL.md](MANUAL.md): the user guide for building sites.
- [design/SPEC.md](design/SPEC.md): how taragen works inside, for contributors.
- [design/](design/README.md): design proposals and the process for
  changing taragen.
- [web/](web/): the taragen website, which is also an example site.

## Development

| Command | What it does |
| --- | --- |
| `make build` | Build `.local/bin/taragen` with the version from git. |
| `make test` | Run `go test ./...`. |
| `make dist` | Cross-compile for darwin, linux, and windows on amd64 and arm64 into `.local/dist/`. |
| `make web-serve` / `make web-build` | Serve or build the example site in `web/`. |
| `make design-check` | Check the design proposals. |

Building needs only Go. The share card rasterizer is resvg compiled to
WebAssembly and embedded in the binary; only changing that module needs Rust
(see its Makefile target).

Changes to taragen's behavior go through a design proposal first; see
[the design process](design/README.md). Agents working in the repository
follow [AGENTS.md](AGENTS.md).

## License

MIT
