---
title: "Show what links to a page"
short: "What links here"
kind: tutorials
num: 9
order: 9
uses: [/docs/manual/backlinks]
---

Add a "Linked from" list to the post layout:

```jsx
{page.backlinks.length > 0 && <aside>
  <h2>Linked from</h2>
  <ul>{page.backlinks.map(p => <li><a href={p.path}>{p.title}</a></li>)}</ul>
</aside>}
```

Only links in page bodies count, not nav or footer links, so the list shows
real references. Set `siteURL` if your posts link to each other with full
URLs. See [Backlinks](/docs/manual/backlinks).
