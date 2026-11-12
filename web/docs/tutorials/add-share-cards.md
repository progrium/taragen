---
title: "Add share cards"
kind: tutorials
num: 8
order: 8
uses: [/docs/manual/defaults-and-globals, /docs/manual/share-cards]
---

1. Put the font files you want on cards in the site, for example
   `_assets/fonts/Inter-Bold.ttf`. The same files can serve your CSS.
2. Add the `wrap` helper from [Defaults and globals](/docs/manual/defaults-and-globals) to
   `_globals.jsx`.
3. Create `_outputs/card.png.jsx`, returning a 1200×630 `<svg>` like the
   example in [Share cards](/docs/manual/share-cards), with `font-family="Inter"`.
4. Point the root layout's `<head>` at it:

   ```jsx
   {page.outputs?.card && <meta property="og:image" content={`${page.siteURL}${page.outputs.card}`} />}
   ```

5. With `taragen serve` running, open `/about.card.png` in the browser. Each
   save reloads it, so you can tune the layout by eye.

If a page should share a different image, set `outputs: { card: false }` and
write your own `og:image` tag for it.
