---
title: "Partials"
kind: manual
num: 9
order: 9
part: "building a site"
---

Partials are reusable pieces of markup you can include in pages, layouts, and
other partials. Any `_name.jsx` or `_name.tmpl` file that isn't one of the
special files above is a partial, and so is any `.jsx` or `.tmpl` file inside an
`_`-prefixed folder. Most sites keep them together in a root `_partials/`
folder, where the files don't need their own underscore.

Include a partial from JSX with its path from the site root, without the
extension:

```jsx
{partial("_partials/footer")}
```

A navigation partial:

```jsx
// _partials/nav.jsx
<ul>
  <li><a href="/">Home</a></li>
  <li><a href="/articles">Articles</a></li>
  <li><a href="/about">About</a></li>
</ul>
```

A partial can be a plain element, as above, or a function. Arguments after the
name are passed to the function:

```jsx
// _partials/card.jsx
(post, showDate = true) =>
<div class="card">
  <a href={post.path}>{post.title}</a>
  {showDate && <time>{post.date}</time>}
</div>
```

```jsx
{pages("articles/**").map(p => partial("_partials/card", p))}
```

Partials see the same globals and `page` as the file that calls them. `.jsx` is
tried first, then `.tmpl`. A missing partial, or one that throws, fails the page
with an error naming it.

In Markdown and Go templates, use `[[ "[[" ]] partial "_partials/footer" ]]`. Template partials
(`.tmpl`) currently receive no helper functions and ignore arguments, so prefer
JSX partials.
