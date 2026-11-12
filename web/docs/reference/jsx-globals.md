---
title: "Built-in JSX globals"
kind: reference
num: 1
order: 1
---

<div class="tw"><table>
<thead><tr><th>Global</th><th>Available in</th><th>Description</th></tr></thead>
<tbody>
<tr><td><code>page</code></td><td class="prose">Pages, layouts, partials, outputs, defaults</td><td class="prose">The current page&#39;s data.</td></tr>
<tr><td><code>data(obj)</code></td><td class="prose">Pages, layouts, outputs</td><td class="prose">Merge <code>obj</code> into page data (in outputs: the output&#39;s settings).</td></tr>
<tr><td><code>partial(name, ...args)</code></td><td class="prose">All JSX</td><td class="prose">Render a partial and insert its HTML.</td></tr>
<tr><td><code>raw(html)</code></td><td class="prose">All JSX</td><td class="prose">Insert an HTML string unescaped.</td></tr>
<tr><td><code>pages(query)</code></td><td class="prose">All JSX</td><td class="prose">List pages by path, pattern, or <code>{ path, where }</code>.</td></tr>
<tr><td><code>params(sets)</code></td><td class="prose">Page templates, as a leading statement</td><td class="prose">Declare the pages a template generates.</td></tr>
<tr><td><code>measureText(text, font)</code></td><td class="prose">All JSX</td><td class="prose">Width of <code>text</code> in pixels, using the site&#39;s fonts.</td></tr>
</tbody>
</table></div>
