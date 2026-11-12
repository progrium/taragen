---
title: "Serve mode and live reload"
kind: manual
num: 24
order: 24
part: "shipping"
---

`taragen serve` is for writing:

- Pages, outputs, generated pages, and cards render when requested, from the
  latest source.
- Drafts are visible and included in listings.
- Aliases answer with `301`; `_redirects` rules answer with their status.
- HTML responses get a small script that reloads the browser when anything in
  the site changes: pages, CSS, images, fonts, `_redirects`, anything outside
  dot-prefixed paths and `_public/`. Non-HTML responses (text outputs, cards,
  feeds) don't get the script.
- A render error shows as a 500 with the error text.
- Paths that aren't pages, outputs, aliases, or redirects are served from disk,
  including underscore paths like `/_assets/...`.

After a change to a page source, a folder, `_outputs/`, a font, or
`_redirects`, the whole site is re-read, so new pages, generated pages,
backlinks, and aliases appear on the next reload.

Requests are rendered one at a time. That's fine for local preview; use
`build` for production.
