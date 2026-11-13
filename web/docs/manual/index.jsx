data({
  title: "Manual",
  dek: "Every taragen feature, in the order you'll need them. The whole manual is also one plain-text file: /manual.txt.",
});
<div class="parts">
  {pages("docs/manual").filter(p => !p.isDir).sort(byOrder).reduce((gs, p) => { const g = gs[gs.length - 1]; if (g && g.part === p.part) g.items.push(p); else gs.push({ part: p.part, items: [p] }); return gs; }, []).map(g =>
    <div class="pt"><span class="m">{g.part}</span><ol class="tl">{g.items.map(p => <li><span class="m">{pad2(p.num)}</span><a href={p.path}>{codeTitle(p.title)}</a></li>)}</ol></div>
  )}
</div>
