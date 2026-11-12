---
title: "Add yearly archive pages"
short: "Yearly archive pages"
kind: tutorials
num: 3
order: 3
---

The year folders from the blog tutorial have no page of their own, so `/articles/2025`
is a 404. A template named `index.jsx` inside a bracketed folder fills them in,
`articles/[year]/index.jsx`:

```jsx
params(() => pages("articles/**").filter(p => p.date).map(p => ({ year: p.date.slice(0, 4) })));
data({ title: `Articles from ${page.params.year}` });
<ul>
  {pages(`articles/${page.params.year}`).reverse().map(p =>
    <li><a href={p.path}>{p.title}</a></li>
  )}
</ul>
```

Each generated year page takes over the year's grouping node and keeps its
posts as `subpages`. Because these pages now have a source, `pages("articles/**")`
includes them; streams that want only posts add `.filter(p => !p.params)`.
