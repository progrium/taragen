// The share card for each page, as a PNG: taragen rasterizes the SVG this
// returns with the site's own fonts. The layout points og:image at it.
const W = 1200, H = 630, pad = 80;
const home = page.path === "." || page.path === "/";
const ink = home ? "#eeeeee" : "#111111", paper = home ? "#0b0b0b" : "#ffffff", soft = home ? "#8a8a8a" : "#777777";
const kind =
  page.kind === "manual" && page.num ? `manual · ${pad2(page.num)}` :
  page.kind === "tutorials" && page.num ? `tutorial · ${pad2(page.num)}` :
  page.kind === "reference" ? "reference" :
  siteTagline;
const title = { family: "Recursive Sans Linear Light", size: 84, weight: 800 };
const text = home ? "Your folder is the website." : plainTitle(page.title ?? siteTitle);
const lines = wrap(text, title, W - 2 * pad).slice(0, 3);
const url = home ? "$ brew install taragen" : siteUrl.replace("https://", "") + page.path;
<svg width={W} height={H} viewBox={`0 0 ${W} ${H}`}>
  <rect width={W} height={H} fill={paper} />
  <text x={pad} y="128" fill={ink} font-family="Silkscreen" font-weight="700" font-size="48">taragen_</text>
  <text x={W - pad} y="124" fill={soft} text-anchor="end" font-family="Recursive Sans Linear Light" font-variation-settings="'MONO' 1" font-size="28">{kind}</text>
  <text fill={ink} font-family={title.family} font-size={title.size} font-weight={title.weight}>
    {lines.map((line, i) => <tspan x={pad} y={300 + i * 96}>{line}</tspan>)}
  </text>
  <text x={pad} y="560" fill={soft} font-family="Recursive Sans Linear Light" font-variation-settings="'MONO' 1" font-size="28">{url}</text>
</svg>
