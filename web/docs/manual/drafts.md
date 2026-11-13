---
title: "Drafts"
kind: manual
num: 14
order: 14
part: "content"
---

A page is a draft when its data has `draft: true`, or its `date` is in the
future. `build` skips drafts entirely: their pages, outputs, and aliases aren't
written, `pages()` doesn't return them, and links to and from them don't count
as backlinks. `serve` shows everything, so you can preview drafts locally.

Future-dated pages publish themselves: the first build on or after their date
includes them.
