---
title: "Non-HTML pages and feeds"
kind: manual
num: 16
order: 16
part: "outputs"
---

A page can produce any text format. Keep the extra extension in the file name,
set its content type and preamble, and turn layouts off:

```jsx
// articles/feed.xml.jsx
data({
  preamble: '<?xml version="1.0" encoding="UTF-8" ?>',
  contentType: "application/rss+xml",
  layout: "",
});
<rss version="2.0">
  <channel>
    <title>{siteTitle}</title>
    <link>{page.siteURL}</link>
    {pages("articles/**").reverse().map(post =>
      <item>
        <title>{post.title}</title>
        <link>{`${page.siteURL}${post.path}`}</link>
        <guid>{`${page.siteURL}${post.path}`}</guid>
        <pubDate>{new Date(post.date).toUTCString()}</pubDate>
        <description>{raw(`<![CDATA[${post.body}]]>`)}</description>
      </item>
    )}
  </channel>
</rss>
```

This builds `_public/articles/feed.xml`. Because escaping is on, titles with `&`
produce valid XML.

Non-HTML pages don't get [extra outputs](/docs/manual/extra-outputs-per-page), aren't
counted as [backlink](/docs/manual/backlinks) sources, and their aliases serve a copy of the
content instead of a redirect stub.
