(content) =>
<html lang="en">
<head>
<meta charset="utf-8" />
<meta name="viewport" content="width=device-width, initial-scale=1" />
<title>{page.title && page.title !== siteTitle ? `${plainTitle(page.title)} · ${siteTitle}` : `${siteTitle} · ${siteTagline}`}</title>
<meta name="description" content={page.description ?? siteDescription} />
<link rel="preload" href="/_assets/fonts/Recursive.woff2" as="font" type="font/woff2" crossorigin="anonymous" />
<link href="/_assets/styles/style.css" rel="stylesheet" />
{page.outputs.txt && <link rel="alternate" type="text/plain" href={page.outputs.txt} />}
{page.outputs.card && <>
<meta property="og:title" content={plainTitle(page.title ?? siteTitle)} />
<meta property="og:image" content={`${siteUrl}${page.outputs.card}`} />
<meta property="og:image:width" content="1200" />
<meta property="og:image:height" content="630" />
<meta name="twitter:card" content="summary_large_image" />
</>}
<script>{raw(`try{var t=localStorage.getItem("theme");if(t)document.documentElement.dataset.theme=t}catch(e){}`)}</script>
</head>
<body>
<div class="site">
  {partial("_partials/header")}
  <main>{content}</main>
  {partial("_partials/footer")}
</div>
<script src="/_assets/site.js" defer={true}></script>
</body>
</html>
