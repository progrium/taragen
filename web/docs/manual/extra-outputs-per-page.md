---
title: "Extra outputs per page"
kind: manual
num: 17
order: 17
part: "outputs"
toc:
  - ["defining-an-output", "Defining an output"]
  - ["what-an-output-template-sees", "What an output template sees"]
  - ["which-pages-get-outputs", "Which pages get outputs"]
  - ["turning-outputs-off", "Turning outputs off"]
  - ["pageoutputs", "`page.outputs`"]
  - ["errors", "Errors"]
---

An **output** is another file made from each page, next to it: a plain-text
version, a JSON summary, a share card image. Outputs are defined by templates in
an `_outputs/` folder.

<a id="defining-an-output"></a>

## Defining an output

An output template is `_outputs/<name>.jsx`. The output's name is the first word
of the file name, and everything after the first `.` (before `.jsx`) is the URL
suffix:

<div class="tw"><table>
<thead><tr><th>File</th><th>Name</th><th>Suffix</th><th>URL for <code>/articles/2021/ngrok</code></th></tr></thead>
<tbody>
<tr><td><code>_outputs/txt.jsx</code></td><td class="prose"><code>txt</code></td><td class="prose"><code>.txt</code></td><td class="prose"><code>/articles/2021/ngrok.txt</code></td></tr>
<tr><td><code>_outputs/json.jsx</code></td><td class="prose"><code>json</code></td><td class="prose"><code>.json</code></td><td class="prose"><code>/articles/2021/ngrok.json</code></td></tr>
<tr><td><code>_outputs/card.png.jsx</code></td><td class="prose"><code>card</code></td><td class="prose"><code>.card.png</code></td><td class="prose"><code>/articles/2021/ngrok.card.png</code></td></tr>
</tbody>
</table></div>

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

<a id="what-an-output-template-sees"></a>

## What an output template sees

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

<div class="tw"><table>
<thead><tr><th>Key</th><th>Meaning</th></tr></thead>
<tbody>
<tr><td><code>contentType</code></td><td class="prose">The <code>Content-Type</code> in <code>serve</code>. Defaults to the type for the suffix&#39;s last extension (<code>text/plain; charset=utf-8</code> for <code>.txt</code>, <code>image/png</code> for <code>.png</code>), or <code>application/octet-stream</code> if unknown.</td></tr>
<tr><td><code>preamble</code></td><td class="prose">Text written before the output. Defaults to none, even for HTML.</td></tr>
<tr><td><code>scale</code></td><td class="prose">For PNG outputs only; see <a href="/docs/manual/share-cards">Share cards</a>.</td></tr>
</tbody>
</table></div>

An output with `contentType: text/html` is formatted like an HTML page.

<a id="which-pages-get-outputs"></a>

## Which pages get outputs

Every page that has a source and is `text/html` gets every output in scope.
Grouping nodes and non-HTML pages like `feed.xml.jsx` get none. A draft's
outputs are drafts too.

Output templates are scoped like layouts: one in `_outputs/` applies to pages in
that folder and below, and for each name, the nearest template wins. So
`logs/_outputs/card.png.jsx` replaces the root `_outputs/card.png.jsx` for pages
under `logs/`. If the nearer template has a different suffix, those pages get
the nearer suffix.

<a id="turning-outputs-off"></a>

## Turning outputs off

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

<a id="pageoutputs"></a>

## `page.outputs`

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

<a id="errors"></a>

## Errors

An output template that throws fails the build, and in `serve` returns a 500
with the error. If an output's URL is the same as another page, output, static
file, alias, or redirect, the build fails naming both. For example, a static
`notes.txt` next to a page `notes` with a `txt` output is a collision.
