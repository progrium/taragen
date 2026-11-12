---
title: "Build a docs section with a sidebar"
short: "Docs sidebar"
kind: tutorials
num: 4
order: 4
---

Put docs pages in `docs/`, with subsections as folders, and give the section a
layout that lists its pages:

```jsx
// docs/_layout.jsx
(content) =>
<>
  <nav class="sidebar">
    <ul>
      {pages("docs").map(p =>
        <li>
          <a href={p.path}>{p.title}</a>
          {p.isDir && p.subpages.length > 0 &&
            <ul>{p.subpages.map(sub => <li><a href={sub.path}>{sub.title}</a></li>)}</ul>}
        </li>
      )}
    </ul>
  </nav>
  <section>
    <h1>{page.title}</h1>
    {content}
  </section>
</>
```

Pages without a `date` sort by slug, so to control the order, prefix file names
(`01-install.md`) or sort in the layout by a key of your own, such as `order`
in frontmatter:

```jsx
pages("docs").sort((a, b) => (a.order ?? 99) - (b.order ?? 99))
```

A subsection index, such as `docs/tutorials/index.jsx`, lists its own pages:

```jsx
data({ title: "Tutorials" });
<ul>{pages("docs/tutorials").map(p => <li><a href={p.path}>{p.title}</a></li>)}</ul>
```

If `docs/` has no index page, send `/docs` straight to the first page with a
line in `_redirects`, instead of writing a redirect page by hand:

```
/docs   /docs/getting-started   302
```
