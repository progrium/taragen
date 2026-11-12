(content) =>
<html>
<head>
<meta charset="UTF-8" />
<link rel="alternate" type="application/rss+xml" title="Taragen RSS Feed" href="/blog/feed.xml"></link>
<link href="/_assets/styles/style.css" rel="stylesheet" type="text/css"></link>
<link href="https://fonts.googleapis.com/css2?family=Inconsolata:wght@200..900&display=swap" rel="stylesheet"></link>
<title>{ page.title } :: Taragen</title>
{page.outputs.card && <>
<meta property="og:image" content={`${siteUrl}${page.outputs.card}`} />
<meta property="og:image:width" content="1200" />
<meta property="og:image:height" content="630" />
<meta name="twitter:card" content="summary_large_image" />
</>}

</head>
<body>
    {partial("_partials/header")}
    <main>
        {content}
    </main>
    {partial("_partials/footer")}
</body>
</html>