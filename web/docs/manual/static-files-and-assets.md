---
title: "Static files and assets"
kind: manual
num: 15
order: 15
part: "content"
---

Any file that isn't a page source and isn't inside a `_` folder is copied to the
same path in `_public/`: `photos/cat.jpg` is served at `/photos/cat.jpg`. Files
next to a page in a page folder (`about/index.md`, `about/team.jpg`) can be
linked relatively.

The root `_assets/` folder is copied verbatim to `_public/_assets/`. Nothing in
it is ever rendered, even `.jsx` or `.md` files, which makes it the place for
CSS, scripts, fonts, and images:

```jsx
<link rel="stylesheet" href="/_assets/style.css" />
```

`_assets/` folders below the root are not copied. Other `_` folders are never
copied in `build`, though `serve` can still serve any file from disk.
