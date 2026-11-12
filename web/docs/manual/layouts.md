---
title: "Layouts"
kind: manual
num: 8
order: 8
part: "building a site"
toc:
  - ["nesting", "Nesting"]
  - ["choosing-a-layout-explicitly", "Choosing a layout explicitly"]
---

A layout wraps pages. A `_layout.jsx` is a function of one argument, the content
to wrap:

```jsx
(content) =>
<html>
  <head>
    <title>{page.title} · {siteTitle}</title>
  </head>
  <body>
    {partial("_partials/header")}
    <main>{content}</main>
    {partial("_partials/footer")}
  </body>
</html>
```

`content` is already-rendered HTML and is inserted as is. `page` is the data of
the page being wrapped.

<a id="nesting"></a>

## Nesting

A layout applies to pages in its folder and every folder below. Layouts nest
nearest first: a page in `articles/2021/` is wrapped by `articles/_layout.jsx`,
and the result is wrapped by the root `_layout.jsx`. Section layouts usually
return a fragment or a `<div>`, and leave `<html>` to the root layout:

```jsx
// articles/_layout.jsx
(content) =>
<article>
  <h1>{page.title}</h1>
  <time>{page.date}</time>
  {content}
</article>
```

<a id="choosing-a-layout-explicitly"></a>

## Choosing a layout explicitly

Set `layout` in a page's data to skip the folder layouts and use a specific one.
The value is a path from the site root, without the extension:

```markdown
---
layout: about/_wide
---
```

`layout: ""` turns layouts off entirely, which is what feeds and other non-HTML
pages want.

A layout can also set `layout` (with `data()` in JSX, or frontmatter in a
template layout). When it does, the remaining folder layouts are dropped and the
named layout is applied next. This lets one layout hand off to a specific chain:

```jsx
// about/_layout.jsx
data({ layout: "_layouts/wide" });
(content) => <div class="about">{content}</div>
```

Go template layouts (`_layout.tmpl`) get the content through `[[ "[[" ]] content ]]`.
