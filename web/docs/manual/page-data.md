---
title: "Page data"
kind: manual
num: 6
order: 6
part: "building a site"
toc:
  - ["where-data-comes-from", "Where data comes from"]
  - ["built-in-keys", "Built-in keys"]
---

Every page has one data object, exposed as `page` in JSX and Go templates and
returned for each page by `pages()`. Frontmatter, `data()` calls, defaults, and
taragen's own keys all write into this same object.

<a id="where-data-comes-from"></a>

## Where data comes from

Data is filled in this order, and later writers win:

1. Taragen seeds `path` and `slug` (and `params` for generated pages), and sets
   `outputs`.
2. `_defaults.jsx` files, from the site root down to the page's folder.
3. The page itself: YAML frontmatter in Markdown and templates, or `data({...})`
   in JSX.
4. Layouts, while rendering: frontmatter in template layouts, `data()` in JSX
   layouts.

After every page is parsed, taragen sets `links` and `backlinks` on every page.

<a id="built-in-keys"></a>

## Built-in keys

Any key you set that isn't listed here is yours: `title`, `description`, `tags`,
and so on flow through untouched. These keys have built-in meaning:

<div class="tw"><table>
<thead><tr><th>Key</th><th>Set by</th><th>Meaning</th></tr></thead>
<tbody>
<tr><td><code>path</code></td><td class="prose">taragen</td><td class="prose">The page&#39;s URL path, such as <code>/articles/2021/ngrok</code>.</td></tr>
<tr><td><code>slug</code></td><td class="prose">taragen</td><td class="prose">The last segment of the path, such as <code>ngrok</code>.</td></tr>
<tr><td><code>src</code></td><td class="prose">taragen</td><td class="prose">The raw page source (for generated pages, the template&#39;s source).</td></tr>
<tr><td><code>body</code></td><td class="prose">taragen</td><td class="prose">The page&#39;s rendered body, before layouts.</td></tr>
<tr><td><code>isDir</code></td><td class="prose">taragen</td><td class="prose">True for folders and <code>index.*</code> files.</td></tr>
<tr><td><code>subpages</code></td><td class="prose">taragen</td><td class="prose">In <code>pages()</code> results: the page&#39;s children, sorted, each with its own <code>subpages</code>.</td></tr>
<tr><td><code>params</code></td><td class="prose">taragen</td><td class="prose">Generated pages only: the parameter values, such as <code>{ year: &#34;2026&#34;, month: &#34;11&#34; }</code>.</td></tr>
<tr><td><code>outputs</code></td><td class="prose">you, then taragen</td><td class="prose">You set it to turn outputs off; taragen replaces it with a map of output name to URL. See <a href="/docs/manual/extra-outputs-per-page">Extra outputs per page</a>.</td></tr>
<tr><td><code>links</code></td><td class="prose">taragen</td><td class="prose">Sorted list of the site pages this page&#39;s body links to.</td></tr>
<tr><td><code>backlinks</code></td><td class="prose">taragen</td><td class="prose">Pages whose bodies link to this page. See <a href="/docs/manual/backlinks">Backlinks</a>.</td></tr>
<tr><td><code>aliases</code></td><td class="prose">you</td><td class="prose">Old URLs that redirect to this page. Taragen replaces it with the normalized list. See <a href="/docs/manual/aliases-and-redirects">Aliases and redirects</a>.</td></tr>
<tr><td><code>siteURL</code></td><td class="prose">you</td><td class="prose">The site&#39;s absolute origin, such as <code>https://example.com</code>. Used to recognize absolute links to your own pages, and handy for building absolute URLs in feeds and meta tags.</td></tr>
<tr><td><code>layout</code></td><td class="prose">you</td><td class="prose">Layout path to use instead of the folder layouts, such as <code>_blog/_layout</code>. <code>&#34;&#34;</code> turns layouts off.</td></tr>
<tr><td><code>date</code></td><td class="prose">you</td><td class="prose"><code>YYYY-MM-DD</code>. Orders listings; a future date makes the page a draft.</td></tr>
<tr><td><code>draft</code></td><td class="prose">you</td><td class="prose"><code>true</code> makes the page a draft.</td></tr>
<tr><td><code>contentType</code></td><td class="prose">you</td><td class="prose">The HTTP <code>Content-Type</code> in <code>serve</code>. Defaults to <code>text/html</code>.</td></tr>
<tr><td><code>preamble</code></td><td class="prose">you</td><td class="prose">Text written before the output. Defaults to <code>&lt;!DOCTYPE html&gt;</code> for HTML.</td></tr>
</tbody>
</table></div>

Keys set by taragen replace whatever a page sets for them. Avoid using `outputs`,
`links`, `backlinks`, `aliases`, or `params` for your own data.
