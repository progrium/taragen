---
title: "Style code blocks"
kind: tutorials
num: 11
order: 11
uses: [/docs/manual/code-highlighting]
---

Fenced code in Markdown comes out with Chroma classes and no colors. Generate a
stylesheet for a theme you like:

```sh
hugo gen chromastyles --style=monokai > _assets/chroma.css
```

or with Chroma's own CLI, and link it from your layout. Or write a few rules
yourself; see [Code highlighting](/docs/manual/code-highlighting).
