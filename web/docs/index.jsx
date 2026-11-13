data({
  title: "The taragen docs",
  layout: "_layout",
  description: "The taragen manual, tutorials and reference.",
});
<div class="docindex">
  <div class="doc wide">
    <div class="byline">docs</div>
    <h1>The taragen docs</h1>
    <p class="dek">One manual that covers every feature in order, tutorials that put features together, and reference tables to look things up.</p>
  </div>
  <div class="keys dx-keys">
    <a class="key solid" href="/docs/manual/install">start reading ⏎</a>
    <a class="key" href="/docs/manual/quick-start">quick start</a>
    <a class="key" href="/manual.txt">manual.txt</a>
  </div>
  <div class="dx">
    <div>
      <h2><a href="/docs/manual">Manual</a><span class="m">{`${pages("docs/manual").filter(p => !p.isDir).sort(byOrder).length} chapters`}</span></h2>
      <p class="lead">Read it front to back the first time. Every chapter stands alone after that.</p>
      <div class="parts">
        {pages("docs/manual").filter(p => !p.isDir).sort(byOrder).reduce((gs, p) => { const g = gs[gs.length - 1]; if (g && g.part === p.part) g.items.push(p); else gs.push({ part: p.part, items: [p] }); return gs; }, []).map(g =>
          <div class="pt"><span class="m">{g.part}</span><div>{g.items.map(p => <a href={p.path}>{plainTitle(p.title)}</a>)}</div></div>
        )}
      </div>
    </div>
    <div class="dx-side">
      <div>
        <h2><a href="/docs/tutorials">Tutorials</a><span class="m">{pages("docs/tutorials").filter(p => !p.isDir).sort(byOrder).length}</span></h2>
        <ol class="tl">{pages("docs/tutorials").filter(p => !p.isDir).sort(byOrder).map(p => <li><span class="m">{pad2(p.num)}</span><a href={p.path}>{p.title}</a></li>)}</ol>
      </div>
      <div>
        <h2><a href="/docs/reference">Reference</a><span class="m">{pages("docs/reference").filter(p => !p.isDir).sort(byOrder).length}</span></h2>
        <ol class="tl">{pages("docs/reference").filter(p => !p.isDir).sort(byOrder).map(p => <li><span class="m">·</span><a href={p.path}>{codeTitle(p.title)}</a></li>)}</ol>
      </div>
    </div>
  </div>
</div>
