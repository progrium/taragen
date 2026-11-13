---
title: "Building and deploying"
kind: manual
num: 23
order: 23
part: "shipping"
---

`taragen build` writes the site to `_public/`:

1. **Clean.** Everything in `_public/` is removed except dot-prefixed entries at
   its top level. A deploy repository checked out at `_public/.git` survives,
   while renamed and deleted pages don't linger.
2. **Assets.** The root `_assets/` is copied to `_public/_assets/`.
3. **Everything else.** Each URL the site has is written once: pages, outputs
   (including PNG cards), generated pages, alias and redirect pages, static
   files, and `_redirects`.

Drafts are skipped throughout.

Any error stops the build with a non-zero exit: a JSX error, a missing partial,
a bad `params()` set, an invalid alias or `_redirects` line, a card with an
unknown font, or two things claiming the same URL. Warnings, such as a wildcard
redirect that can't be expanded, are printed but don't fail the build.

Builds are deterministic: the same sources produce the same bytes, including
attribute order, listing order, and PNG cards. That makes `_public/` diffs
reviewable.

To deploy, upload `_public/` to any static host. Clean URLs
(`/about` → `/about/index.html`) and files beside folders
(`/about.txt` next to `/about/`) work on GitHub Pages, Netlify, S3, and similar
hosts without configuration.
