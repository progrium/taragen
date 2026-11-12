---
title: "Highlight the current page in navigation"
short: "Current page in nav"
kind: tutorials
num: 5
order: 5
aliases: [/docs/tutorials/active-nav-state]
---

`page` in a partial is the page being rendered, so a nav partial can compare
each link with `page.path`:

```jsx
// _partials/nav.jsx
() => {
  const link = (href, text) =>
    <a class={page.path === href || page.path.startsWith(href + "/") ? "selected" : ""} href={href}>{text}</a>;
  return (
    <ul>
      <li>{link("/", "Home")}</li>
      <li>{link("/articles", "Articles")}</li>
      <li>{link("/about", "About")}</li>
    </ul>
  );
}
```

The `startsWith` check keeps "Articles" selected on every post. An empty class
is left out of the output, so unselected links have no `class` attribute. Style
it in your CSS:

```css
nav .selected { font-weight: bold; }
```

Include it in a layout with `{partial("_partials/nav")}`.
