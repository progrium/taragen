---
title: "Format dates"
kind: tutorials
num: 6
order: 6
aliases: [/docs/tutorials/date-formatter-utility]
---

`date` is a `YYYY-MM-DD` string. Add a formatter to your root `_globals.jsx`:

```js
formatDate = (date, format = "MM/dd/yyyy") => {
  const d = new Date(date);
  const map = {
    yyyy: String(d.getUTCFullYear()),
    MM: String(d.getUTCMonth() + 1).padStart(2, "0"),
    dd: String(d.getUTCDate()).padStart(2, "0"),
  };
  return format.replace(/yyyy|MM|dd/g, m => map[m]);
};
```

and use it in any JSX:

```jsx
<time datetime={page.date}>{formatDate(page.date, "dd.MM.yyyy")}</time>
```

The UTC methods matter: `new Date("2025-01-01")` is midnight UTC, and local
time methods would show the previous day west of Greenwich.

To use a global in a Markdown page, wrap it in a partial, such as
`_partials/today.jsx` containing `<span>{formatDate(new Date())}</span>`, and
include it with `[[ "[[" ]] partial "_partials/today" ]]`.
