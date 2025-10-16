---
title: Implement an RSS feed
---

We've added a page data option called `preamble` that allows you to insert a tag at the start of the layout, to better support RSS feeds.

As you can see in the example, you need to explicitly set it to `layout = ""`, so it does not use one of the existing layouts.

In this example, we have a blog directory with a `feed.xml.jsx` file inside of it, with the following:

```

data({
    preamble: '<?xml version="1.0" encoding="UTF-8" ?>',
    contentType: "application/rss+xml",
    layout: "",
});

<rss xmlns:content="http://purl.org/rss/1.0/modules/content/" xmlns:wfw="http://wellformedweb.org/CommentAPI/" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:atom="http://www.w3.org/2005/Atom" xmlns:sy="http://purl.org/rss/1.0/modules/syndication/" xmlns:slash="http://purl.org/rss/1.0/modules/slash/" version="2.0">
    <channel>
        <title>{siteTitle}</title>
        <link>{siteUrl}</link>
        <atom:link href={`${siteUrl}${page.path}`} rel="self" type="application/rss+xml"/>
        <description>{siteDescription}</description>
        <language>en</language>
        <generator>Taragen</generator>
        {pages("blog/*").map(post => (
            <item>
                <title>{post.title}</title>
                <link>`{{ safeHTML `{`${siteUrl}${post.path}`}` }}`</link>
                <guid isPermaLink="false">{`${siteUrl}${post.path}`}</guid>
                <description>{post.description}</description>
                <content:encoded>
                    {`<![CDATA[${post.body}]]>`}
                </content:encoded>
                <pubDate>{new Date(post.date).toUTCString()}</pubDate>
            </item>
        ))}
    </channel>
</rss>
```
