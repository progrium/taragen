---
title: "Commands"
kind: manual
num: 3
order: 3
part: "getting started"
---

Both commands work on the current directory as the site root.

<div class="tw"><table>
<thead><tr><th>Command</th><th>What it does</th></tr></thead>
<tbody>
<tr><td><code>taragen serve</code></td><td class="prose">Serves the site at <code>http://localhost:8088</code>, renders pages on request, shows drafts, and live reloads the browser when files change.</td></tr>
<tr><td><code>taragen build</code></td><td class="prose">Cleans <code>_public/</code> and writes the whole site into it. Drafts are left out. Also available as <code>taragen gen</code> and <code>taragen generate</code>.</td></tr>
</tbody>
</table></div>

The port (`8088`) and output directory (`_public`) are fixed.

`build` exits with a non-zero status on any error, so it is safe to use in CI
and deploy scripts.
