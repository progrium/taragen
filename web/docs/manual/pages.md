---
title: "Pages"
kind: manual
num: 5
order: 5
part: "building a site"
---

A page source is a `.jsx`, `.md`, or `.tmpl` file. Each format produces an HTML
**body**, which is then wrapped by layouts.

<div class="tw"><table>
<thead><tr><th>Format</th><th>Best for</th><th>Page data from</th></tr></thead>
<tbody>
<tr><td>JSX (<code>.jsx</code>)</td><td class="prose">Index pages, listings, anything with logic</td><td class="prose">A leading <code>data({...})</code> call</td></tr>
<tr><td>Markdown (<code>.md</code>)</td><td class="prose">Writing: articles, docs, notes</td><td class="prose">YAML frontmatter</td></tr>
<tr><td>Go template (<code>.tmpl</code>)</td><td class="prose">Simple templated pages and layouts</td><td class="prose">YAML frontmatter</td></tr>
</tbody>
</table></div>

A JSX page:

```jsx
data({ title: "Articles" });
<section>
  <h1>{page.title}</h1>
  <ul>
    {pages("articles/**").reverse().map(a =>
      <li><a href={a.path}>{a.title}</a> <small>{a.date}</small></li>
    )}
  </ul>
</section>
```

A Markdown page:

```markdown
---
title: Building your own Ngrok in 130 lines
date: 2021-03-14
tags: [go, networking]
---

Ngrok is a tunnel...
```
