---
title: "Defaults and globals"
kind: manual
num: 7
order: 7
part: "building a site"
---

Both are JSX files that run as plain scripts. Every bare assignment at the top
level becomes an export, so write `siteTitle = "My site"`, not `export` or
`const`:

```js
// _globals.jsx
siteTitle = "My site";
greeting = <strong>Hello!</strong>;
wrap = (text, font, width) => text.split(" ").reduce((lines, word) => {
  const last = lines[lines.length - 1];
  if (last && measureText(`${last} ${word}`, font) <= width) lines[lines.length - 1] = `${last} ${word}`;
  else lines.push(word);
  return lines;
}, []);
```

**`_globals.jsx`** defines names that every JSX file in its folder and below can
use: pages, layouts, partials, and output templates. Use it for site constants
and helper functions. Globals files run from the root down, so a nested
`_globals.jsx` sees and can override what its parents defined. Nested globals
are merged, so to hide a parent's global in a section, set it to `undefined` in
that section's `_globals.jsx`.

Since globals files are JSX, a global can be markup, like `greeting` above, and
used anywhere as `{greeting}`.

Globals are for JSX. Markdown and Go template pages can't read them directly;
to use one there, wrap it in a JSX partial and include the partial with
`[[ "[[" ]] partial "_partials/name" ]]`.

**`_defaults.jsx`** sets default page data for every page in its folder and
below. Its exports are merged into each page's data before the page's own
frontmatter or `data()`, so pages can override them. `page` is available in it,
so defaults can depend on the page:

```js
// _defaults.jsx
siteURL = "https://example.com";

// articles/_defaults.jsx
aliases = [page.path.replace(/^\/articles/, "/blog")];
```

Built-in globals (`page`, `data`, `partial`, `raw`, `pages`, `params`,
`measureText`) are added after your globals, so they always win. Don't define
globals with those names.

Every JSX render runs in a fresh JavaScript VM, and globals are re-run for each
one. Pages can't leak state into each other, so don't try to share mutable state
through globals.
