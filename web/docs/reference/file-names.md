---
title: "File naming summary"
kind: reference
num: 4
order: 4
---

<div class="tw"><table>
<thead><tr><th>Pattern</th><th>Meaning</th></tr></thead>
<tbody>
<tr><td><code>.name</code></td><td class="prose">Ignored</td></tr>
<tr><td><code>_name/</code></td><td class="prose">Special folder, never a page or copied (except root <code>_assets/</code>)</td></tr>
<tr><td><code>_layout.jsx</code> / <code>_layout.tmpl</code></td><td class="prose">Layout</td></tr>
<tr><td><code>_globals.jsx</code></td><td class="prose">Globals</td></tr>
<tr><td><code>_defaults.jsx</code></td><td class="prose">Default page data</td></tr>
<tr><td><code>_outputs/&lt;name&gt;[.ext].jsx</code></td><td class="prose">Output template</td></tr>
<tr><td><code>_name.jsx</code> / <code>_name.tmpl</code></td><td class="prose">Partial</td></tr>
<tr><td><code>_redirects</code> (root)</td><td class="prose">Redirect rules</td></tr>
<tr><td><code>name.jsx</code>, <code>name.md</code>, <code>name.tmpl</code></td><td class="prose">Page at <code>/name</code></td></tr>
<tr><td><code>name/index.*</code></td><td class="prose">Page at <code>/name</code></td></tr>
<tr><td><code>name.xml.jsx</code></td><td class="prose">Page at <code>/name.xml</code></td></tr>
<tr><td><code>[param].jsx</code>, <code>[param]/</code></td><td class="prose">Page template</td></tr>
<tr><td>anything else</td><td class="prose">Static file</td></tr>
</tbody>
</table></div>
