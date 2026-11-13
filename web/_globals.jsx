siteTitle = "taragen"
siteTagline = "static sites from one binary"
siteDescription = "Taragen is a fast, minimal static site generator in a single binary."
siteUrl = "https://progrium.github.io/taragen"
repoUrl = "https://github.com/progrium/taragen"

// Docs pages sort by the order key in their frontmatter.
byOrder = (a, b) => (a.order ?? 99) - (b.order ?? 99)

pad2 = (n) => String(n).padStart(2, "0")

// Titles like "Listing pages with `pages()`" keep their code formatting.
plainTitle = (t) => (t ?? "").replaceAll("`", "")
codeTitle = (t) => (t ?? "").split("`").map((s, i) => i % 2 ? <code>{s}</code> : s)

// A short label for a docs page, as backlinks and the sidebar show it.
docLabel = (p) =>
  p.kind === "manual" ? `manual ${pad2(p.num)} · ${plainTitle(p.short ?? p.title)}` :
  p.kind === "tutorials" ? `tutorial ${pad2(p.num)} · ${plainTitle(p.short ?? p.title)}` :
  p.kind === "reference" ? `reference · ${plainTitle(p.title)}` :
  plainTitle(p.title ?? p.path)

// Greedy word wrap with the same measurements the share card renders with.
wrap = (text, font, width) => text.split(" ").reduce((lines, word) => {
  const last = lines[lines.length - 1];
  if (last && measureText(`${last} ${word}`, font) <= width) lines[lines.length - 1] = `${last} ${word}`;
  else lines.push(word);
  return lines;
}, []);
