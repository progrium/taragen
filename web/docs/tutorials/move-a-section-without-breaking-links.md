---
title: "Move a section without breaking links"
short: "Move a section"
kind: tutorials
num: 10
order: 10
uses: [/docs/manual/aliases-and-redirects]
---

To rename `blog/` to `articles/`:

1. Move the folder.
2. Add `articles/_defaults.jsx` so every page answers at its old URL too:

   ```js
   aliases = [page.path.replace(/^\/articles/, "/blog")];
   ```

3. Build. `/blog/2021/ngrok` gets a redirect page to `/articles/2021/ngrok`,
   `/blog` redirects to `/articles`, and the old feed URL `/blog/feed.xml` keeps
   serving the feed itself, so subscribers don't notice. Old pages are gone from
   `_public/` because every build starts clean.

On a host that reads `_redirects`, you can add `/blog/* /articles/:splat` too,
for real 301s. See [Aliases and redirects](/docs/manual/aliases-and-redirects).
