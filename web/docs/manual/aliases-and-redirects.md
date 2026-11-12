---
title: "Aliases and redirects"
kind: manual
num: 21
order: 21
part: "site structure"
toc:
  - ["page-aliases", "Page aliases"]
  - ["what-an-alias-produces", "What an alias produces"]
  - ["the-_redirects-file", "The `_redirects` file"]
  - ["conflicts", "Conflicts"]
---

When a page moves, keep its old URL working with an **alias**. For redirects
that aren't tied to a page, use a **`_redirects`** file.

<a id="page-aliases"></a>

## Page aliases

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

<a id="what-an-alias-produces"></a>

## What an alias produces

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

<a id="the-_redirects-file"></a>

## The `_redirects` file

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

<a id="conflicts"></a>

## Conflicts

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
