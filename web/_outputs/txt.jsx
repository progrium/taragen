// The plain text version of each page: its title, then its source.
// Docs pages are generated from MANUAL.md by tools/docs, which adds anchors
// and HTML tables; this turns those back into plain Markdown.
data({ contentType: "text/plain; charset=utf-8" });
const unhtml = (s) => s
  .replace(/<code>(.*?)<\/code>/g, "`$1`")
  .replace(/<a href="([^"]*)">(.*?)<\/a>/g, "[$2]($1)")
  .replace(/<\/?b>/g, "**")
  .replace(/<[^>]+>/g, "")
  .replace(/&lt;/g, "<").replace(/&gt;/g, ">").replace(/&#34;|&quot;/g, '"').replace(/&#39;/g, "'").replace(/&amp;/g, "&");
const table = (html) => {
  const rows = [...html.matchAll(/<tr>(.*?)<\/tr>/gs)].map(m => [...m[1].matchAll(/<t[hd][^>]*>(.*?)<\/t[hd]>/gs)].map(c => unhtml(c[1])));
  if (!rows.length) return html;
  const line = (r) => `| ${r.join(" | ")} |`;
  return [line(rows[0]), line(rows[0].map(() => "---")), ...rows.slice(1).map(line)].join("\n");
};
const text = page.kind
  ? page.src
      .replace(/^---\n[\s\S]*?\n---\n+/, "")
      .replace(/<a id="[^"]*"><\/a>\n\n/g, "")
      .replace(/<div class="tw"><table>[\s\S]*?<\/table><\/div>/g, table)
      .replaceAll('[[ "[[" ]]', "[[")
  : page.src;
`${plainTitle(page.title ?? page.path)}\n\n${text}`
