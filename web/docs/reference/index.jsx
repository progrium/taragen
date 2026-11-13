data({
  title: "Reference",
  dek: "Tables to look things up in. Each entry links to the chapter that explains it.",
});
<ol class="tl">{pages("docs/reference").filter(p => !p.isDir).sort(byOrder).map(p => <li><span class="m">·</span><a href={p.path}>{codeTitle(p.title)}</a></li>)}</ol>
