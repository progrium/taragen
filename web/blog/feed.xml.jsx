data({
    layout: "_partials/xml",
    contentType: "application/rss+xml",
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
                <link>{`${siteUrl}${post.path}`}</link>
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