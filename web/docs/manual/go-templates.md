---
title: "Go templates"
kind: manual
num: 12
order: 12
part: "formats"
---

`.tmpl` files use Go's `text/template` with `[[ "[[" ]]` and `]]` as delimiters, so they
don't clash with `{...}` in content and code. Output isn't auto-escaped.
Frontmatter works as in Markdown.

<div class="tw"><table>
<thead><tr><th>Helper</th><th>Meaning</th></tr></thead>
<tbody>
<tr><td><code>[[ "[[" ]] content ]]</code></td><td class="prose">In a layout, the wrapped content; in a page, the body.</td></tr>
<tr><td><code>[[ "[[" ]] page &#34;title&#34; ]]</code></td><td class="prose">A string value from page data.</td></tr>
<tr><td><code>[[ "[[" ]] partial &#34;_partials/footer&#34; ]]</code></td><td class="prose">Includes a partial.</td></tr>
<tr><td><code>safeHTML</code></td><td class="prose">Returns its argument unchanged; kept for compatibility.</td></tr>
</tbody>
</table></div>

The `page` helper only handles string values, and templates have no `pages`
helper, so lists like `tags`, `backlinks`, and listings are for JSX.
