(content) =>
<html>
<head>
<meta charset="UTF-8" />
<link rel="alternate" type="application/rss+xml" title="Taragen RSS Feed" href="/blog/feed.xml"></link>
<link href="/style.css" rel="stylesheet"></link>
<link href="https://fonts.googleapis.com/css2?family=Figtree:ital,wght@0,400;0,500;0,600;0,700;0,800;0,900;1,400&display=swap" rel="stylesheet" />
<title>{ page.title } :: Taragen</title>

</head>
<body>
    {partial("_partials/header")}
    {content}
    {partial("_partials/footer")}
</body>
</html>