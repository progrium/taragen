---
title: Active state for navigation
tags: [jsx, layouts]
---

In your JSX page file (or layout file), reference a nav partial:
```jsx
{partial("_nav")}
```

Add the following to a CSS file:
```css
.selected {
	font-weight: bold;
}
```

Create a `nav.jsx` file with:
```jsx
() => {
  const navLink = (href, text) => {
    let className = "";
    if (page.path === href) {
      className = "selected";
    }
    return <a class={className} href={href}>{text}</a>;
  }

  return (
    <ul>
      <li>{navLink("/", "Home")}</li>
      <li>{navLink("/blog", "Blog")}</li>
      <li>{navLink("/about", "About")}</li>
      <li>{navLink("/projects/foo", "Foo")}</li>
    </ul>
  )
}
```
