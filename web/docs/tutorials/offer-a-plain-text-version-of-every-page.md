---
title: "Offer a plain-text version of every page"
short: "Plain-text pages"
kind: tutorials
num: 7
order: 7
uses: [/docs/manual/extra-outputs-per-page]
---

Add `_outputs/txt.jsx`:

```jsx
data({ contentType: "text/plain; charset=utf-8" });
`${page.title ?? page.date}\n\n${page.src}`
```

Every HTML page now has a `.txt` twin, `/about` beside `/about.txt`, holding
its title and raw source. Link it from the root layout:

```jsx
{page.outputs?.txt && <a href={page.outputs.txt}>Plain text</a>}
```

Turn it off where it makes no sense with `outputs: { txt: false }` in a page's
frontmatter, or for a whole folder in its `_defaults.jsx`. See
[Extra outputs per page](/docs/manual/extra-outputs-per-page).
