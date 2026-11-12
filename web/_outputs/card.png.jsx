// The share card for each page, as a PNG: taragen rasterizes the SVG this
// returns with the site's own fonts. The layout points og:image at it.
const W = 1200, H = 630, pad = 80;
const title = { family: "Perfectly Nineties", size: 88, weight: 700 };
const lines = wrap(page.title ?? siteTitle, title, W - 2 * pad).slice(0, 3);
<svg width={W} height={H} viewBox={`0 0 ${W} ${H}`}>
  <rect width={W} height={H} fill="#fffaf0" />
  <text x={pad} y="120" font-family="Perfectly Nineties" font-weight="600" font-size="36">{siteTitle}</text>
  <text font-family={title.family} font-size={title.size} font-weight={title.weight}>
    {lines.map((line, i) => <tspan x={pad} y={260 + i * 100}>{line}</tspan>)}
  </text>
  <rect x={pad} y="520" width={W - 2 * pad} height="6" fill="#111" />
  <text x={pad} y="580" font-family="Perfectly Nineties" font-weight="600" font-size="28">{siteUrl.replace("https://", "") + (page.path === "." ? "/" : page.path)}</text>
</svg>
