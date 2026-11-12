---
title: "Add tag pages"
kind: tutorials
num: 2
order: 2
uses: [/docs/manual/generated-pages]
---

Tag posts in frontmatter with `tags: [go, networking]`. Then one template makes
a page for every tag in use, `tags/[tag].jsx`:

```jsx
params(() => [...new Set(pages("articles/**").flatMap(p => p.tags ?? []))].map(tag => ({ tag })));
data({ title: `Tagged “${page.params.tag}”` });
<>
  <h1>{page.title}</h1>
  <ul>
    {pages({ path: "articles/**", where: { tags: page.params.tag } }).reverse().map(p =>
      <li><a href={p.path}>{p.title}</a></li>
    )}
  </ul>
</>
```

A tag index, `tags/index.jsx`, lists the generated pages, since page bodies run
after every page exists:

```jsx
data({ title: "Tags" });
<ul>{pages("tags").map(t => <li><a href={t.path}>{t.slug}</a></li>)}</ul>
```

Link each post's tags from the post layout:

```jsx
{page.tags?.map(t => <a class="tag" href={`/tags/${t}`}>{t}</a>)}
```

To give one tag a hand-written page, add `tags/go.md`; it replaces the
generated `/tags/go` and every other tag keeps its generated page. See
[Generated pages](/docs/manual/generated-pages).
