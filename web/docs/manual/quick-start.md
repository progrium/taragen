---
title: "Quick start"
kind: manual
num: 2
order: 2
part: "getting started"
aliases: [/docs/getting-started, /docs/install, /start]
---

Make a folder for your site, and in it an `index.jsx`:

```jsx
<html>
  <body>
    <h1>Hello World</h1>
  </body>
</html>
```

From that folder, run:

```sh
taragen serve
```

Open `http://localhost:8088`. Edit the file and the browser reloads by itself.

Add a Markdown page, `about.md`:

```markdown
---
title: About
---

This site is built with **taragen**.
```

It appears at `/about`. Now give every page the same frame with a root
`_layout.jsx`:

```jsx
(content) =>
<html>
  <head>
    <meta charset="utf-8" />
    <title>{page.title ?? "My site"}</title>
  </head>
  <body>
    <nav><a href="/">Home</a> <a href="/about">About</a></nav>
    <main>{content}</main>
  </body>
</html>
```

Remove the `<html>` and `<body>` from `index.jsx`, since the layout provides
them now. When you're ready to publish:

```sh
taragen build
```

The finished site is in `_public/`, ready to upload to any static host.
