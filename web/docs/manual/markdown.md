---
title: "Markdown"
kind: manual
num: 11
order: 11
part: "formats"
---

Markdown pages start with optional YAML frontmatter between `---` lines, then
Markdown content. Raw HTML in Markdown is allowed and passed through.

Markdown is first run through the Go template engine, so `[[ "[[" ]] ... ]]` actions
work in Markdown:

```markdown
---
title: Notes
---

This page lives at [[ "[[" ]] page "path" ]].
```

This also means a literal `[[ "[[" ]]` anywhere in Markdown, even in a code block, is
read as a template action. Write it as `[[ "[[" ]] "[[ "[[" ]]" ]]`, or use JSX for pages that
need a lot of it.

Fenced code blocks with a language are syntax highlighted; see
[Code highlighting](/docs/manual/code-highlighting).
