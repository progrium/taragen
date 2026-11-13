// The docs frame: a sidebar of all three sections, then the page with its
// place in the section, previous and next links, and what links to it.
(content) => {
  const docPages = (kind) => pages(`docs/${kind}`).filter(p => !p.isDir).sort(byOrder);
  const kinds = [["manual", "manual"], ["tutorials", "tutorials"], ["reference", "reference"]];
  const kind = page.kind;
  const list = kind ? docPages(kind) : [];
  const i = list.findIndex(p => p.path === page.path);
  const prev = i > 0 ? list[i - 1] : null;
  const next = i >= 0 && i < list.length - 1 ? list[i + 1] : null;
  const num = (p) => p.kind === "reference" ? "·" : pad2(p.num);

  const item = (p) =>
    <li>
      <a href={p.path} aria-current={p.path === page.path ? "page" : ""}><span class="num">{num(p)}</span><span>{plainTitle(p.short ?? p.title)}</span></a>
      {p.path === page.path && page.toc?.length > 1 &&
        <div class="toc">{page.toc.map(([id, title]) => <a href={`#${id}`}>{plainTitle(title)}</a>)}</div>}
    </li>;

  const section = ([k, label]) => {
    const ps = docPages(k);
    const open = k === kind;
    const groups = k === "manual"
      ? ps.reduce((gs, p) => { const g = gs[gs.length - 1]; if (g && g.part === p.part) g.items.push(p); else gs.push({ part: p.part, items: [p] }); return gs; }, [])
      : [{ part: null, items: ps }];
    return <div class="grp">
      <div class="gh"><a href={`/docs/${k}`} aria-current={page.path === `/docs/${k}` ? "page" : ""}>{label}</a><span class="n">{ps.length}</span></div>
      {open && groups.map(g => <>
        {g.part && <div class="part">{g.part}</div>}
        <ol>{g.items.map(item)}</ol>
      </>)}
    </div>;
  };

  const byline =
    page.kind === "manual" && page.num ? `manual · ${page.num} of ${list.length} · ${page.part}` :
    page.kind === "tutorials" && page.num ? `tutorial · ${pad2(page.num)} of ${list.length}` :
    page.kind === "reference" && page.num ? "reference" :
    "docs";
  const uses = (page.uses ?? []).map(path => docPages("manual").find(p => p.path === path)).filter(Boolean);
  const refs = page.backlinks.filter(ref => ref.num).sort(byOrder);
  const editUrl = page.kind ? `${repoUrl}/blob/main/MANUAL.md` : `${repoUrl}/tree/main/web/docs`;

  return <div class="docs">
    <aside class="side" aria-label="Docs contents">
      <details open={true}>
        <summary>contents</summary>
        {kinds.map(section)}
        <div class="whole">[ <a href="/manual.txt">manual.txt</a> ]</div>
      </details>
    </aside>

    <article class="doc">
      <div class="byline">{byline}</div>
      <h1>{codeTitle(page.title)}</h1>
      {page.dek && <p class="dek">{page.dek}</p>}
      {uses.length > 0 && <div class="uses"><span class="m soft">uses</span>{uses.map(p => <a class="tag" href={p.path}>{`${pad2(p.num)} ${plainTitle(p.short ?? p.title)}`}</a>)}</div>}
      <div class={`doc-body${page.kind === "manual" && page.num ? " numbered" : ""}`} style={page.kind === "manual" && page.num ? `--ch: "${page.num}."` : ""}>
        {content}
      </div>
      {(prev || next) &&
        <nav class="pn" aria-label="Previous and next">
          {prev ? <a href={prev.path}><span class="k">{`← ${num(prev)}`}</span><span class="t">{plainTitle(prev.title)}</span></a> : <span></span>}
          {next ? <a class="next" href={next.path}><span class="k">{`${num(next)} →`}</span><span class="t">{plainTitle(next.title)}</span></a> : <span></span>}
        </nav>}
      <div class="post-foot">
        {refs.length > 0 &&
          <div class="lf"><span>linked from:</span>{refs.map(ref => <a href={ref.path}>{docLabel(ref)}</a>)}</div>}
        <div>[ {page.outputs.txt && <><a href={page.outputs.txt}>txt</a> · </>}<a href={editUrl}>edit on github</a>{page.outputs.card && <> · <a href={page.outputs.card}>card</a></>} ]</div>
      </div>
    </article>
  </div>;
}
