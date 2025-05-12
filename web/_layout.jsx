(content) =>
<html>
<head>
<meta charset="UTF-8" />
<link rel="alternate" type="application/rss+xml" title="Taragen RSS Feed" href="/blog/feed.xml"></link>
<link href="/_assets/styles/style.css" rel="stylesheet" type="text/css"></link>
<link href="https://fonts.googleapis.com/css2?family=Inconsolata:wght@200..900&display=swap" rel="stylesheet"></link>
<title>{ page.title } :: Taragen</title>

</head>
<body>
    {partial("_partials/header")}
    <main>
        {content}
    </main>
    {partial("_partials/footer")}
</body>
</html>