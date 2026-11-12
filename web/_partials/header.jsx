<header class="s-head">
  <a class="s-name" href="/" aria-label="taragen home">taragen<span class="cur" aria-hidden="true">_</span></a>
  <span class="s-tag">{siteTagline}</span>
  <nav class="s-nav" aria-label="Site">
    {[["/docs", "docs"], ["/docs/tutorials", "tutorials"], ["/docs/reference", "reference"], [repoUrl, "github"]].map(([href, text]) => {
      const here = page.path === href || page.path.startsWith(href + "/");
      const inOther = href === "/docs" && (page.path.startsWith("/docs/tutorials") || page.path.startsWith("/docs/reference"));
      return <a href={href} aria-current={here && !inOther ? "page" : ""}>{text}</a>;
    })}
  </nav>
</header>
