---
title: "Errors and troubleshooting"
kind: reference
num: 99
order: 99
---

<div class="tw"><table>
<thead><tr><th>Symptom</th><th>Cause and fix</th></tr></thead>
<tbody>
<tr><td>Page shows <code>&amp;amp;</code> or <code>&amp;lt;p&amp;gt;</code> where HTML should be</td><td class="prose">You inserted an HTML string as text. Wrap it in <code>raw()</code>: <code>{raw(post.body)}</code>.</td></tr>
<tr><td><code>TypeError</code> from JSX</td><td class="prose">A child or attribute is an unsupported value, like an object as a child. Render a field of it instead.</td></tr>
<tr><td>&#34;partial not found&#34;</td><td class="prose">The path in <code>partial()</code> is from the site root and without extension: <code>partial(&#34;_partials/footer&#34;)</code>.</td></tr>
<tr><td>A page appears in <code>serve</code> but not in <code>build</code></td><td class="prose">It&#39;s a draft: <code>draft: true</code> or a future <code>date</code>.</td></tr>
<tr><td><code>[[ "[[" ]]</code> in a Markdown page breaks it</td><td class="prose">Markdown runs through the template engine first. Write it as <code>[[ "[[" ]] &#34;[[ "[[" ]]&#34; ]]</code>, or use JSX for that page.</td></tr>
<tr><td>URL collision error</td><td class="prose">Two things produce one URL (a page and static file, an output and a page, two aliases). The error names both; rename or remove one.</td></tr>
<tr><td>A tag page isn&#39;t generated</td><td class="prose"><code>params()</code> only sees real, non-draft pages (in <code>build</code>). Check the source page has the tag and isn&#39;t a draft.</td></tr>
<tr><td>A card fails with an unknown font family</td><td class="prose">Add the font file to the site (e.g. <code>_assets/fonts/</code>) and use its family name exactly. System fonts aren&#39;t used.</td></tr>
<tr><td>A link isn&#39;t in <code>backlinks</code></td><td class="prose">Links in layouts don&#39;t count; links to aliases, outputs, and static files don&#39;t count; absolute links need <code>siteURL</code>.</td></tr>
<tr><td>A relative link counts toward the wrong page</td><td class="prose">Relative links resolve against <code>&lt;path&gt;/</code> for clean-URL pages, as in the browser. Prefer root-relative links (<code>/docs/setup</code>).</td></tr>
<tr><td>A wildcard redirect works in <code>serve</code> but not on the host</td><td class="prose">It couldn&#39;t be expanded into redirect pages (see the build warning). Use a host that reads <code>_redirects</code>, or point it at a path with <code>:splat</code>.</td></tr>
<tr><td>Code isn&#39;t colored</td><td class="prose">Taragen emits classes, not colors. Add Chroma CSS to your stylesheet.</td></tr>
<tr><td>Logs print twice</td><td class="prose">Bodies that call <code>pages()</code> are parsed twice so listings see every page.</td></tr>
</tbody>
</table></div>
