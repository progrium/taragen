---
title: "Writing JSX"
kind: manual
num: 10
order: 10
part: "formats"
toc:
  - ["pages-data-and-multiple-roots", "Pages, data, and multiple roots"]
  - ["values-in-jsx", "Values in JSX"]
  - ["escaping-and-raw", "Escaping and `raw()`"]
  - ["whitespace-and-pre", "Whitespace and `<pre>`"]
  - ["output-formatting", "Output formatting"]
---

Taragen's JSX is a server-side template language. Elements turn straight into
HTML strings; there is no React, no components with state, no hooks, and nothing
runs in the browser. Write `class`, not `className`.

<a id="pages-data-and-multiple-roots"></a>

## Pages, data, and multiple roots

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
it. In a [generated page template](/docs/manual/generated-pages), a `params(...)` call can
sit alongside `data()` in these leading statements.

<a id="values-in-jsx"></a>

## Values in JSX

Children can be elements, strings, numbers, or arrays (flattened, so
`.map(...)` works). `null`, `undefined`, `true`, and `false` render nothing, so
conditionals work:

```jsx
{page.draft && <span class="badge">Draft</span>}
{page.tags?.map(t => <a href={`/tags/${t}`}>{t}</a>)}
```

Attribute values can be:

<div class="tw"><table>
<thead><tr><th>Value</th><th>Result</th></tr></thead>
<tbody>
<tr><td>string</td><td class="prose"><code>name=&#34;value&#34;</code>; an empty string omits the attribute</td></tr>
<tr><td>number</td><td class="prose">Printed as JS prints it</td></tr>
<tr><td><code>true</code> / <code>false</code></td><td class="prose">Bare attribute (<code>&lt;input disabled&gt;</code>) / omitted</td></tr>
<tr><td>object</td><td class="prose">Serialized as a <code>style</code> string: <code>style={{ color: &#34;red&#34;, &#34;font-size&#34;: &#34;2em&#34; }}</code></td></tr>
<tr><td><code>null</code> / <code>undefined</code></td><td class="prose">Omitted</td></tr>
</tbody>
</table></div>

Anything else, such as an object as a child or a function as an attribute,
throws a `TypeError` and fails the render.

Attributes are written in the order you wrote them, and the output is identical
on every build, so diffs of `_public/` show only real changes. Void elements
like `<img>`, `<meta>`, and `<br>` are self-closed.

<a id="escaping-and-raw"></a>

## Escaping and `raw()`

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

<a id="whitespace-and-pre"></a>

## Whitespace and `<pre>`

JSX normally collapses whitespace, but newlines inside `<pre>` blocks are kept,
so code samples written in JSX keep their lines.

<a id="output-formatting"></a>

## Output formatting

HTML output (`text/html`) is passed through an HTML formatter that normalizes
indentation. Other content types are written exactly as rendered.
