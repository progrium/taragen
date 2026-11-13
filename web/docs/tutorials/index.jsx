data({
  title: "Tutorials",
  dek: "Common tasks from start to finish. Each one links back to the manual chapters it uses.",
});
<ol class="tl">{pages("docs/tutorials").filter(p => !p.isDir).sort(byOrder).map(p => <li><span class="m">{pad2(p.num)}</span><a href={p.path}>{p.title}</a></li>)}</ol>
